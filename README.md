# gh-atlas

## Installation
Install the `gh` CLI - see the [installation](https://github.com/cli/cli#installation)

Install this extension:

```sh
gh extension install ariga/gh-atlas
```

add permissions to add workflow files
```bash
gh auth refresh -s write:packages,workflow
```
   
## Development
clone the repo:
```bash
git clone https://github.com/ariga/gh-atlas
```
add extension locally:
```bash
gh extension install .
```
see changes in your code as you develop:
```bash
go build && gh atlas
```
