package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ankitpokhrel/jira-cli/pkg/md"
)

const (
	threadAddRouteV1 = "/gateway/api/graphql/pq/f5fb37d86837b6f99ab87528af1b0f2454e8a7d5c1b17ec1571aa8c47c48b3f9"
	threadAddQueryV1 = "useSaveCommentRelayAddCommentMutation"
	// replySourceByteLimit bounds synchronous Markdown parsing, which contexts cannot interrupt.
	replySourceByteLimit = 256 << 10
)

// ErrReplyOutcomeUnknown means a reply may have been created. Callers must retain
// the draft and must not retry the mutation automatically.
var ErrReplyOutcomeUnknown = errors.New("reply outcome unknown; check Jira before retrying")

// ReplyToIssueThread posts one reply to a freshly verified unrestricted native
// root in a software/business Cloud project. Call only after explicit user Post.
func (c *Client) ReplyToIssueThread(ctx context.Context, key, rootCommentID, body string) error {
	if err := c.replyToIssueThread(ctx, key, rootCommentID, body); err != nil {
		return fmt.Errorf("jira: reply to native comment thread: %w", err)
	}
	return nil
}

// replyToIssueThread validates the draft and freshly checks target privacy before
// one mutation; success requires read-back confirmation, not just an accepted POST.
func (c *Client) replyToIssueThread(ctx context.Context, key, rootCommentID, body string) error {
	if !threadIssueKey.MatchString(key) || !validReplyID(rootCommentID) {
		return errors.New("invalid issue key or root comment ID")
	}
	if len(body) > replySourceByteLimit {
		return fmt.Errorf("reply Markdown source exceeds %d bytes", replySourceByteLimit)
	}
	if strings.TrimSpace(body) == "" {
		return errors.New("reply body must not be empty or whitespace")
	}
	r, err := c.newThreadReader()
	if err != nil {
		return err
	}
	timeout := threadTotalTimeout
	if c.timeout > 0 {
		timeout = min(timeout, c.timeout)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("before reply preparation: %w", err)
	}
	doc, err := md.ToADF(body)
	if err != nil {
		return fmt.Errorf("encode reply Markdown: %w", err)
	}
	if doc == nil || doc.DocType != "doc" || doc.Version != 1 || len(doc.Content) == 0 {
		return errors.New("reply Markdown produced an empty or invalid ADF document")
	}
	issueID, rootNodeID, err := r.replyTarget(ctx, key, rootCommentID)
	if err != nil {
		return fmt.Errorf("validate reply target: %w", err)
	}
	variables := map[string]any{
		"input": map[string]any{
			"issueId": issueID, "threadParentId": rootCommentID,
			"content":         map[string]any{"version": 1, "jsonValue": doc},
			"permissionLevel": nil, "eventOccurredAt": nil,
		},
		"connections":   []string{"client:" + rootNodeID + ":::::JiraPlatformComment:__commentRepliesListRelay_repliesPaginated_connection"},
		"isRootComment": false, "includeRedactedContent": false,
		threadMentionVarV1: true, threadCommentsVarV1: false,
	}
	payload, err := json.Marshal(map[string]any{"operationName": threadAddQueryV1, "variables": variables})
	if err != nil {
		return errors.New("could not encode reply request")
	}
	if len(payload) > threadResponseLimit {
		return errors.New("reply request size limit exceeded")
	}
	if r.requests > threadRequestLimit-2 {
		return errors.New("insufficient request budget for reply confirmation")
	}
	id, err := r.postReply(ctx, payload)
	if err != nil {
		return err
	}
	if id == rootCommentID {
		return unknownReply(id, errors.New("mutation returned the selected root instead of a new reply"))
	}
	if err := r.confirmReply(ctx, key, rootCommentID, id); err != nil {
		return unknownReply(id, err)
	}
	return nil
}

// validReplyID accepts only canonical positive decimal IDs, excluding aliases
// such as leading-zero forms that could undermine identity comparisons.
func validReplyID(id string) bool {
	number, err := strconv.ParseUint(id, 10, 64)
	return err == nil && number > 0 && strconv.FormatUint(number, 10) == id
}

// replyTarget revalidates issue identity, supported project type, native root
// membership, and unrestricted visibility before any write is permitted.
func (r *threadReader) replyTarget(ctx context.Context, key, rootID string) (string, string, error) {
	body, err := r.get(ctx, baseURLv3+"/issue/"+key+"?fields=project", false)
	if err != nil {
		return "", "", err
	}
	var issue struct {
		ID, Key string
		Fields  struct {
			Project struct {
				ProjectTypeKey string `json:"projectTypeKey"`
			} `json:"project"`
		} `json:"fields"`
	}
	if err := decodeReplyJSON(body, &issue); err != nil {
		return "", "", err
	}
	if !validReplyID(issue.ID) || !strings.EqualFold(issue.Key, key) {
		return "", "", errors.New("missing or inconsistent reply issue identity")
	}
	switch issue.Fields.Project.ProjectTypeKey {
	case "software", "business":
	default:
		return "", "", errors.New("native replies support software/business projects only; service or unknown project contexts are unsupported")
	}
	issueID, err := r.bootstrap(ctx, key, issue.ID)
	if err != nil {
		return "", "", err
	}
	parents, err := r.parents(ctx, issueID, key)
	if err != nil {
		return "", "", err
	}
	root, exists := parents[rootID]
	if !exists || root.parentID != "" {
		return "", "", errors.New("selected comment is not a verified native root of this issue")
	}
	comment, err := r.replyComment(ctx, key, rootID)
	if err != nil {
		return "", "", err
	}
	if len(comment.ParentID) != 0 && !bytes.Equal(bytes.TrimSpace(comment.ParentID), []byte("null")) {
		return "", "", errors.New("selected root now has a parent or invalid linkage")
	}
	if err := comment.unrestricted(); err != nil {
		return "", "", err
	}
	return issueID, root.nodeID, nil
}

// replyComment retains raw linkage and visibility values so unexpected shapes
// cannot be mistaken for an unrestricted comment or a verified parent.
type replyComment struct {
	ID         string          `json:"id"`
	ParentID   json.RawMessage `json:"parentId"`
	Visibility json.RawMessage `json:"visibility"`
	Properties *[]struct {
		Key   string          `json:"key"`
		Value json.RawMessage `json:"value"`
	} `json:"properties"`
}

// replyComment fetches privacy properties as well as linkage and verifies the
// response identity, both for preflight checks and post-mutation confirmation.
func (r *threadReader) replyComment(ctx context.Context, key, id string) (*replyComment, error) {
	body, err := r.get(ctx, baseURLv3+"/issue/"+key+"/comment/"+id+"?expand=properties", false)
	if err != nil {
		return nil, err
	}
	var comment replyComment
	if err := decodeReplyJSON(body, &comment); err != nil {
		return nil, err
	}
	if comment.ID != id {
		return nil, errors.New("missing or inconsistent REST reply comment identity")
	}
	return &comment, nil
}

// unrestricted fails closed on restricted or unknown visibility and missing
// privacy metadata, preventing a reply from silently changing the audience.
func (c *replyComment) unrestricted() error {
	if len(c.Visibility) != 0 && !bytes.Equal(bytes.TrimSpace(c.Visibility), []byte("null")) {
		return errors.New("restricted or unknown comment visibility is unsupported for native replies")
	}
	if c.Properties == nil {
		return errors.New("comment privacy properties were not returned; refusing an unverified reply")
	}
	seen := make(map[string]bool)
	for _, property := range *c.Properties {
		if property.Key == "" || seen[property.Key] || len(property.Value) == 0 {
			return errors.New("missing or duplicate comment privacy property")
		}
		seen[property.Key] = true
		if property.Key != "sd.public.comment" {
			continue
		}
		if !publicCommentProperty(property.Value) {
			return errors.New("internal or unknown comment privacy is unsupported for native replies")
		}
	}
	return nil
}

// publicCommentProperty accepts only an explicit internal:false object, rejecting
// extra or duplicate fields instead of guessing at unfamiliar privacy metadata.
func publicCommentProperty(raw json.RawMessage) bool {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	open, err := decoder.Token()
	if err != nil || open != json.Delim('{') {
		return false
	}
	key, err := decoder.Token()
	if err != nil || key != "internal" {
		return false
	}
	var internal *bool
	if err := decoder.Decode(&internal); err != nil || internal == nil || *internal {
		return false
	}
	closingToken, err := decoder.Token()
	return err == nil && closingToken == json.Delim('}')
}

// postReply attempts the mutation once; failures after dispatch are uncertain
// unless an authentication or authorization rejection proves no reply was accepted.
func (r *threadReader) postReply(ctx context.Context, payload []byte) (string, error) {
	query := url.Values{"operation": {threadAddQueryV1}}
	body, attempted, err := r.request(ctx, http.MethodPost, threadAddRouteV1+"?"+query.Encode(), payload, false)
	if err != nil {
		var status threadHTTPError
		if !attempted || errors.As(err, &status) && (status.status == http.StatusUnauthorized || status.status == http.StatusForbidden) {
			return "", err
		}
		return "", unknownReply("", err)
	}
	return decodeReplyResult(body)
}

// decodeReplyResult requires consistent success metadata and a new comment ID;
// partial or contradictory results retain uncertainty rather than invite a retry.
func decodeReplyResult(body []byte) (string, error) {
	var response struct {
		Errors []json.RawMessage `json:"errors"`
		Data   struct {
			Jira struct {
				AddComment *struct {
					Success *bool           `json:"success"`
					Errors  json.RawMessage `json:"errors"`
					Comment struct {
						ID string `json:"commentId"`
					} `json:"comment"`
				} `json:"addComment"`
			} `json:"jira"`
		} `json:"data"`
	}
	if err := decodeReplyJSON(body, &response); err != nil {
		return "", unknownReply("", err)
	}
	payload := response.Data.Jira.AddComment
	if payload == nil {
		return "", unknownReply("", errors.New("missing GraphQL reply result"))
	}
	id := payload.Comment.ID
	if !validReplyID(id) {
		id = ""
	}
	var businessErrors []json.RawMessage
	if len(payload.Errors) == 0 || threadDecode(payload.Errors, &businessErrors) != nil || payload.Success == nil || len(response.Errors) != 0 {
		return "", unknownReply(id, errors.New("missing, malformed, or partial GraphQL reply result"))
	}
	if !*payload.Success && id == "" {
		return "", errors.New("jira rejected the native reply; business error details redacted")
	}
	if !*payload.Success || len(businessErrors) != 0 || id == "" {
		return "", unknownReply(id, errors.New("inconsistent GraphQL reply success or comment identity"))
	}
	return id, nil
}

// confirmReply verifies the created comment's parent and visibility by REST
// read-back before callers may treat the mutation as confirmed and clear a draft.
func (r *threadReader) confirmReply(ctx context.Context, key, rootID, id string) error {
	comment, err := r.replyComment(ctx, key, id)
	if err != nil {
		return fmt.Errorf("confirm reply: %w", err)
	}
	parent := string(bytes.TrimSpace(comment.ParentID))
	if strings.HasPrefix(parent, `"`) {
		if err := threadDecode(comment.ParentID, &parent); err != nil {
			return err
		}
	}
	if !validReplyID(parent) || parent != rootID {
		return errors.New("created reply has missing, invalid, or unexpected parent linkage")
	}
	return comment.unrestricted()
}

// decodeReplyJSON rejects duplicate keys before decoding because ordinary JSON
// unmarshalling could let a later value hide privacy or mutation failure metadata.
func decodeReplyJSON(body []byte, out any) error {
	// Duplicate keys must not hide a restriction, parent, or failed mutation.
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := uniqueReplyJSON(decoder); err != nil {
		return errors.New("invalid or ambiguous reply metadata JSON")
	}
	return threadDecode(body, out)
}

// uniqueReplyJSON walks nested objects with a separate key set for each scope,
// detecting ambiguity before any metadata is trusted for a write decision.
func uniqueReplyJSON(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, container := token.(json.Delim)
	if !container {
		return nil
	}
	seen := make(map[string]bool)
	for decoder.More() {
		if delimiter == '{' {
			key, err := decoder.Token()
			if err != nil {
				return err
			}
			name, valid := key.(string)
			if !valid || seen[name] {
				return errors.New("duplicate or invalid object key")
			}
			seen[name] = true
		}
		if err := uniqueReplyJSON(decoder); err != nil {
			return err
		}
	}
	_, err = decoder.Token()
	return err
}

// unknownReply preserves the uncertainty sentinel and underlying cause, adding
// a returned comment ID when available so users can check Jira before retrying.
func unknownReply(id string, cause error) error {
	if id != "" {
		return fmt.Errorf("%w (returned comment %s): %w", ErrReplyOutcomeUnknown, id, cause)
	}
	return fmt.Errorf("%w: %w", ErrReplyOutcomeUnknown, cause)
}
