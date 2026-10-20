---
name: jira-cli
description: Use Jira-CLI to search, inspect, create, or update Jira issues and related project data. Covers machine-readable output, configuration metadata, command discovery, and safe execution.
---

# Jira CLI

Use this skill when operating Jira through the `jira` command. Start with `jira --help`, then inspect the relevant command and subcommand help (for
example, `jira issue list --help`) before using them. The installed CLI's help is the source of truth for its available commands, flags, and syntax; it also
contains command-specific examples. This skill adds agent-oriented safety guidance and useful workflow patterns that may not be obvious from the flags.

## Prefer machine-readable output

List commands are interactive by default. When reading data programmatically, prefer an output format intended for parsing:

```sh
# JSON
jira issue list --raw

# Plain text, optionally selecting fields and omitting headers
jira issue list --plain --columns key,summary --no-headers

# CSV
jira issue list --csv
```

Use `--raw` when supported and JSON is needed. Otherwise, use `--plain` with `--columns` and `--no-headers` to minimize parsing ambiguity.

## Configuration and fields

The default configuration file is:

```text
$XDG_CONFIG_HOME/.jira/.config.yml
```

When `XDG_CONFIG_HOME` is unset, it is:

```text
~/.config/.jira/.config.yml
```

The configuration file can be used as a data source. It contains the configured project, issue types, and custom field definitions. In particular, inspect
`issue.types` and `issue.fields.custom` before constructing `create`, `edit`, or `--custom` commands.

Use `--config/-c` or `JIRA_CONFIG_FILE` to select another configuration file.

This metadata is created during `jira init` and, over time, can become out of sync with Jira. Refreshing the configuration with `jira init` changes local
configuration. NEVER do it without obtaining explicit confirmation first.

For custom fields, use `--custom key=value`. The key is derived from the configured field name in lowercase with words separated by hyphens (for example,
`Story Points` becomes `story-points`). The expected value format depends on the field type; array values are comma-separated. Only use fields present in the
configuration, and inspect their schema rather than guessing the format.

Do not guess a field name, ID, allowed value, or issue type when it is missing or inconsistent. Instead, use another authorized source of truth, such as the Jira
UI or API. Some error messages contain relevant data; for instance, when `jira issue move` fails due to an invalid transition state, it lists available
states in the error output. If everything fails, ask the user.

## Useful JQL patterns

Use `jira issue list --jql` (`-q`) when the built-in filters are insufficient. For example, raw JQL can search across multiple projects, use a saved filter, or
approximate a backlog query:

```sh
# Issues in either project
jira issue list -q "project IN (PRJ1, PRJ2)"

# Issues matching a saved Jira filter
jira issue list -q "filter='my saved filter'"

# Backlog-style issues (adapt statuses and issue types to the project workflow)
jira issue list -s~Done -q "sprint is empty and issuetype not in (Epic, Sub-task)" --order-by rank --reverse
```

When the JQL itself contains a project condition, it takes precedence over `--project` (`-p`); do not expect `-p` to further restrict that query.

## Safe execution

- NEVER expose `JIRA_API_TOKEN` or config credentials; `--debug` may print auth headers.
- ALWAYS confirm destructive actions and externally visible changes the user did not explicitly request.
- Read an issue before updating it when its current state matters. Verify consequential or uncertain updates.
- Use `jira <command> [<subcommand>] --help` for flags, requirements, and examples. Supply explicit arguments and use `--no-input` when supported.
- Cache useful discoveries for the session.
