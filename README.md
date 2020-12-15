<h1 align="center">JiraCLI</h1>

<div>
    <p align="center">
        <img alt="Build" src="https://img.shields.io/github/workflow/status/ankitpokhrel/jira-cli/build?style=flat-square" />
        <img alt="GO Report-card" src="https://goreportcard.com/badge/github.com/ankitpokhrel/jira-cli?style=flat-square" />
    </p>
    <p align="center">
        <i>Jira command line to help me with my frequent Jira chores</i>
    </p>
    <img align="center" alt="TusPHP Demo" src=".github/assets/demo.gif" /><br/><br/>
    <p align="center">:construction: This project is still a work in progress :construction:</p><br/>
</div>

Jira UI is terrible! It is slow, buggy, and doesn't even load on occasions. Fortunately, Jira API seems to have a decent response time.
Even though not everything is available in the public API, things can be hacked around to port routine tasks to the CLI and avoid
the UI as much as possible.

This tool mostly focuses on issue search and navigation at the moment. However, it will include some daily operations like issue creation,
updating a ticket status, and so on.

### Installation
Install the runnable binary to your `$GOPATH/bin`.

```sh
$ go install github.com/ankitpokhrel/jira-cli

# optionally, rename the binary
$ mv $GOPATH/bin/jira-cli $GOPATH/bin/jira
```

Releases and other installation options will be available later.

### Shell completion
Check `jira completion --help` for more info on setting up a bash/zsh shell completion. 

### Getting started
1. [Get a Jira API token](https://id.atlassian.com/manage-profile/security/api-tokens) and export it to your shell as a `JIRA_API_TOKEN` variable.
    Add it to your shell configuration file, for instance, `$HOME/.bashrc`, so that the variable is always available.
2. Run `jira init` to generate a config file required for the tool.

### Usage
The tool currently comes with an issue, epic, and sprint explorer. The flags are [POSIX-compliant](https://www.gnu.org/software/libc/manual/html_node/Argument-Syntax.html).
You can combine available flags in any order to create a unique query. For example, the command below will give you high priority issues created this month
with status `To Do` that are assigned to you and has a label `backend` :exploding_head:.
```sh
$ jira issue -yHigh -s"To Do" --created month -lbackend -a$(jira me)
```

The lists are displayed in an interactive UI by default. 
- Use arrow keys or `j, k, h, l` characters to navigate through the list. 
- Press `ENTER` to open the selected issue in the browser.
- In an explorer view, press `w` to toggle focus between the sidebar and the contents screen.
- Press `q` / `ESC` / `CTRL+C` to quit.

##### Notes
The tool:
- Doesn't yet support pagination and returns 100 records by default.
- Only returns 25 recent sprints. 
- Is only tested with the latest Jira cloud. 

Check some examples/use-cases below.

#### Issue

<details><summary>List recent issues</summary>

```
$ jira issue
```
</details>

<details><summary>List issues that I am watching</summary>

```sh
$ jira issue -w
```
</details>

<details><summary>List issues assigned to me</summary>

```sh
$ jira issue -a$(jira me)
```
</details>

<details><summary>List issues assigned to a user and are reported by another user</summary>

```sh
$ jira issue -a"User A" -e"User B"
```
</details>

<details><summary>List issues assigned to me is of high priority and is open</summary>

```sh
$ jira issue -a$(jira me) -yHigh -sopen
```
</details>

<details><summary>List issues assigned to no one and are created this week</summary>

```sh
$ jira issue -ax --created week
```
</details>

<details><summary>List issues created within an hour and updated in the last 30 minutes :stopwatch:</summary>

```sh
$ jira issue --created -1h --updated -30m
```
</details>

<details><summary>Give me issues that are of high priority, is in progress, was created this month, and has given labels :fire:</summary>

```sh
$ jira issue -yHigh -s"In Progress" --created month -lbackend -l"high prio"
```
</details>

<details><summary>Wait, what was that ticket I opened earlier today? :tired_face:</summary>

 ```sh
 $ jira issue --history
 ```
</details>

<details><summary>What was the first issue I ever reported on the current board? :thinking:</summary>

```sh
$ jira issue -e$(jira me) --reverse
```
</details>

<details><summary>What was the first bug I ever fixed in the current board? :beetle:</summary>

```sh
$ jira issue -a$(jira me) -tBug sDone -rFixed --reverse
```
</details>

<details><summary>What issues did I report this week? :man_shrugging:</summary>

```sh
$ jira issue -e$(jira me) --created week
```
</details>

<details><summary>Am I watching any tickets in project XYZ? :monocle_face:</summary>

```sh
$ jira issue -w -pXYZ
```
</details>

#### Epic

Epics are displayed in an explorer view by default. You can output the results in a table view using the `--list` flag.
When viewing epic issues, you can use all filters available for the issue command.

<details><summary>List epics</summary>

```sh
$ jira epic

// or, in a list view
$ jira epic --list
```
</details>

<details><summary>List epics reported by me and are open</summary>

```sh
$ jira epic -e$(jira me) -sOpen
```
</details>

<details><summary>List issues in an epic</summary>

```sh
$ jira epic KEY-1

// list all issue in an epic KEY-1 that is unassigned and has a high priority
$ jira epic KEY-1 -ax -yHigh
```
</details>

<details><summary>List issues in an epic that is unassigned and has a high priority</summary>

```sh
$ jira epic KEY-1 -ax -yHigh
```
</details>

#### Sprint

Sprints are displayed in an explorer view by default. You can output the results in a table view using the `--list` flag.
When viewing sprint issues, you can use all filters available for the issue command. The tool only shows 25 recent sprints.

<details><summary>List sprints</summary>

```sh
$ jira sprint

// or, in a list view
$ jira sprint --list
```
</details>

<details><summary>List future and active sprints</summary>

```sh
$ jira sprint --state future,active
```
</details>

<details><summary>List issues in a sprint</summary>

```sh
// you can get sprint id with `jira sprint` or `jira sprint --list`
$ jira sprint SPRINT_ID
```
</details>

<details><summary>List high priority issues in a sprint are assigned to me</summary>

```sh
$ jira sprint SPRINT_ID -yHigh -a$(jira me)
```
</details>

#### Other commands

<details><summary>Navigate to the project</summary>

```sh
$ jira open
```
</details>

<details><summary>Navigate to the issue</summary>

```sh
$ jira open KEY-1
```
</details>

<details><summary>List all projects you have access to</summary>

```sh
$ jira project
```
</details>

<details><summary>List all boards in a project</summary>

```sh
$ jira board
```
</details>

### Scripts

Often times, you may want to use the output of the command to do something cool. However, the default interactive UI might not allow you to do that.
The tool comes with the `--plain` flag that displays results in a simple layout that can then be manipulated from the shell script.    

Some example scripts are listed below.

###### Tickets created per day this month

Using a similar trick like the one below, you can get the number of tickets created per month and compare it with last year's data.

```bash
#!/usr/bin/env bash

tickets=$(jira issues --created month --plain | rev | awk '{print $2}' | rev | sed "1 d" | awk -F'-' '{print $3}' | sort -n | uniq -c)

echo "${tickets}" | while IFS=$'\t' read -r line; do
  day=$(echo "${line}" | awk '{print $2}')
  count=$(echo "${line}" | awk '{print $1}')

  printf "Day #%s: %s\n" "${day}" "${count}"
done

# Output
Day #01: 19
Day #02: 10
Day #03: 21
Day #04: 15
Day #05: 17
...
```

###### Number of tickets per sprint

```bash
#!/usr/bin/env bash

sprints=$(jira sprint --list --plain | cut -f1 -f2 | sed "1 d")

echo "${sprints}" | while IFS=$'\t' read -r id name; do
  count=$(jira sprint "${id}" --list --plain | wc -l)

  printf "%10s: %3d\n" "${name}" $((count - 1))
done

# Output
Sprint 5:   58
Sprint 4:   52
Sprint 3:   55
Sprint 2:   40
Sprint 1:   30
...
```

###### Number of unique assignee per sprint

```bash
#!/usr/bin/env bash

sprints=$(jira sprint --list --plain | cut -f1 -f2 | sed "1 d")

echo "${sprints}" | while IFS=$'\t' read -r id name; do
  count=$(jira sprint "${id}" --list --plain | tr '\t' ',' | sed 's/,\{2,\}/,/g' | cut -d',' -f4 | sed "1 d" | sort -n | uniq | wc -l)

  printf "%10s: %3d\n" "${name}" $((count - 1))
done

# Output
Sprint 5:   6
Sprint 4:   6
Sprint 3:   5
Sprint 2:   4
Sprint 1:   3
```

### Future improvements
- [ ] Issue creation.
- [ ] Ability to view issue details.
- [ ] Possibility to change issue status.
- [ ] Pagination support.
- [ ] Historical data can be cached locally for faster execution.

### Development
1. Clone the repo.
   ```sh
   $ git clone git@github.com:ankitpokhrel/jira-cli.git
   ```

2. Make changes, build the binary, and test your changes.
   ```sh
   $ make install
   ```   

3. Run linter and tests before submitting a PR.
   ```sh
   $ make lint
   $ make test
   ```
