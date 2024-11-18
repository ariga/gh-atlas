# gh-atlas

A GitHub CLI extension for managing [Atlas](https://github.com/ariga/atlas) workflows.

## Getting started

To quickly set up [Atlas Cloud](https://atlasgo.cloud) CI for your repository, follow these steps:

1. Install the official `gh` CLI:
  ```sh
  brew install gh
  ```
  For other systems, see the [installation](https://github.com/cli/cli#installation) instructions.
2. Install this extension:
  ```sh
  gh extension install ariga/gh-atlas
  ```
3. Login to GitHub:
  ```sh
  gh auth login
  ```
4. Setting up Atlas Cloud for your repository requires creating new GitHub Actions workflows.  
  To do this, you need add the following permissions to your GitHub CLI:
  ```sh
  gh auth refresh -s write:packages,workflow
  ```
5. Use the `init-action` command to set up Atlas Cloud for your repository:
  ```sh
  gh atlas init-action
  ```
  This will create a new workflow in your repository, which will run on every push to the your mainline branch.  
  You can customize the workflow by editing the `.github/workflows/ci-atlas.yml` file.
