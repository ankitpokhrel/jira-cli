# JiraCLI for Agents

This document contains agent-specific guidance. The [README](./README.md) is
the authoritative reference for installation, authentication, commands, and
flags. Ignore its images, animations, and interactive-navigation instructions
unless the task concerns the terminal UI.

## Prefer machine-readable output

List commands are interactive by default. When reading data programmatically,
prefer an output format intended for parsing:

```sh
# JSON
jira issue list --raw

# Plain text, optionally selecting fields and omitting headers
jira issue list --plain --columns key,summary --no-headers

# CSV
jira issue list --csv
```

Use `--raw` when supported and JSON is needed. Otherwise, use `--plain` with
`--columns` and `--no-headers` to minimize parsing ambiguity. See
[Scripts](./README.md#scripts) for examples.

## Configuration and fields

The default configuration file is:

```text
$XDG_CONFIG_HOME/.jira/.config.yml
```

When `XDG_CONFIG_HOME` is unset, it is:

```text
~/.config/.jira/.config.yml
```

The configuration file can be used as a data source. The configuration contains
the configured project, issue types, and custom field definitions. In particular,
inspect `issue.types` and `issue.fields.custom` before constructing `create`,
`edit`, or `--custom` commands.

Use `--config/-c` or `JIRA_CONFIG_FILE` to select another configuration file.

This metadata is created during `jira init` and, over time, can become out of sync with Jira.
Refreshing the configuration with `jira init` changes local configuration. NEVER do it without
obtaining an explicit confirmation first.

Do not guess a field name, ID, allowed value, or issue type when it is missing or inconsistent.
Instead, use another authorized source of truth, such as the Jira UI or API. Some error messages
contain relevant data; for instance, when `jira issue move` command fails due to invalid transition
state, it lists available states in the error output. If everything fails, ask the user.

## Safe execution

- NEVER disclose `JIRA_API_TOKEN` or credentials from the configuration file. The `--debug` flag
prints HTTP request details, including authorization headers.
- ALWAYS request confirmation before destructive actions like deleting issues or making an externally visible
change not explicitly requested by the user.
- Avoid assuming flags and use `jira <command> [<subcommand>] --help` for command-specific flags and requirements.
- Supply explicit arguments to avoid prompts; use `--no-input` when supported.
- Cache useful discoveries for the session to avoid redundant lookups.
