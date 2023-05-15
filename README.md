# gh-atlas
GitHub CLI extension for managing [Atlas](https://github.com/ariga/atlas) workflows.

## Installation
Install the `gh` CLI - see the [installation](https://github.com/cli/cli#installation)

Install this extension

```sh
gh extension install ariga/gh-atlas
```

## Usage

add permissions to add workflow files
```bash
gh auth refresh -s write:packages,workflow
```

create pull request with atlas CI workflow by running
```bash
gh atlas init-action
```

for more information run
```bash
gh atlas init-action -h
```
   
## Development
clone the repo:
```bash
git clone https://github.com/ariga/gh-atlas
```
add extension locally
```bash
cd gh-atlas && gh extension install .
```
see changes in your code as you develop
```bash
go build && gh atlas
```
