# Contributing Guidelines

Everybody is glad to see you contributing to this project. In this document, we will provide you some guidelines in order to help get your contribution accepted.

## Reporting an issue

### Issues

When you find a bug in terraform-provider-cartography, it should be reported using [GitHub issues](https://github.com/ankek/terraform-provider-cartography/issues). Please define key information like your Operating System (OS), Terrafrom version and finally the Terafrom Provider versions you are using.

### Issue Types

There are 6 types of labels, they can be used for issues or PRs:

- `enhancement`: These track specific feature requests and ideas until they are completed. They can evolve from a `specification` or they can be submitted individually depending on the size.
- `specification`: These track issues with a detailed description, this is like a proposal.
- `bug`: These track bugs with the code
- `docs`: These track problems with the documentation (i.e. missing or incomplete)
- `maintenance`: These tracks problems, update and migration for dependencies / third-party tools
- `refactoring`: These tracks internal improvement with no direct impact on the product
- `need review`: this status must be set when you feel confident with your submission
- `in progress`: some important change has been requested on your submission, so you can toggle from `need review` to `in progress`
- `under discussion`: it's time to take a break, think about this submission and try to figure out how we can implement this or this

## Submit a contribution

### Setup your git repository

If you want to contribute to an existing issue, you can start by _forking_ this repository, then clone your fork on your machine.

```shell
$ git clone https://github.com/<your-username>/terraform-provider-cartography.git
$ cd terraform-provider-cartography
```

In order to stay updated with the upstream, it's highly recommended to add `ankek/terraform-provider-cartography` as a remote upstream.

```shell
$ git remote add upstream https://github.com/ankek/terraform-provider-cartography
```

Do not forget to frequently update your fork with the upstream.

```shell
$ git fetch upstream --prune
$ git rebase upstream/master
```

### Play with the codebase

#### Build from sources
Terraform Cartography is a GO Language project, Go must be installed and configured on your machine (really ?). We currently support GO 1.25+ and go `modules` as dependency manager. You can simply pull all necessaries dependencies by running an initial.

```shell
$ make build
```

This basically builds `terraform-provider-cartography` with the current sources.

#### Add new icons

Terraform Cartography use embedded icons and store it in binary file.

### Code Architecture

In this section we'll explain the main packages if you want to contribute/read/understand how terraform-provider-cartography works.



