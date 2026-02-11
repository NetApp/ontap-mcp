# Contributing
Thank you for your interest in contributing to the ONTAP-MCP project! 🎉

We appreciate that you want to take the time to contribute! Please follow these steps before submitting your PR.

# Creating a Pull Request

1. Please search [existing issues](https://github.com/NetApp/ontap-mcp/issues) to determine if an issue already exists for what you intend to contribute.
2. If the issue does not exist, [create a new one](https://github.com/NetApp/ontap-mcp/issues/new) that explains the bug or feature request.
3. Let us know in the issue that you plan on creating a pull request for it. This helps us to keep track of the pull request and make sure there isn't duplicate effort.
4. Before creating a pull request, write up a brief proposal in the issue describing what your change would be and how it would work so that others can comment.
5. It's better to wait for feedback from someone on NetApp's ONTAP-MCP development team before writing code. We don't have an SLA for our feedback, but we will do our best to respond in a timely manner (at a minimum, to give you an idea if you're on the right track and that you should proceed, or not).
6. Sign and submit NetApp's Contributor License Agreement. You must sign and submit the [Corporate Contributor License Agreement (CCLA)](https://netapp.tap.thinksmart.com/prod/Portal/ShowWorkFlow/AnonymousEmbed/3d2f3aa5-9161-4970-997d-e482b0b033fa) in order to contribute. 
7. Make sure you specify `NetApp/ontap-mcp` or `https://github.com/NetApp/ontap-mcp` for the **Project Name**.
8. If you've made it this far, have written the code that solves your issue, and addressed the review comments, then feel free to create your pull request.

Important: **NetApp will NOT look at the PR or any of the code submitted in the PR if the CCLA is not on file with NetApp Legal.**

# ONTAP-MCP Team's Commitment
While we truly appreciate your efforts on pull requests, we **cannot** commit to including your PR in the ONTAP-MCP project. Here are a few reasons why:

* There are many factors involved in integrating new code into this project, including things like: 
    * support for a wide variety of NetApp backends
    * proper adherence to our existing and/or upcoming architecture
    * sufficient functional and/or scenario tests across all backends
    * etc.

In other words, while your bug fix or feature may be perfect as a standalone patch, we have to ensure that the changes work in all use cases, configurations, backends and across our support matrix.

* The ONTAP-MCP team must plan our resources to integrate your code into our code base and CI platform, and depending on the complexity of your PR, we may or may not have the resources available to make it happen in a timely fashion. We'll do our best.

* Sometimes a PR doesn't fit into our future plans or conflicts with other items on the roadmap. It's possible that a PR you submit doesn't align with our upcoming plans, thus we won't be able to use it. It's not personal.

Thank you for considering to contribute to the ONTAP-MCP project!

# Changelog

The changelog is one of the most important ways we communicate with stakeholders. As such, it needs careful attention and focus. 

Creating the changelog is a mixture of auto-generated commands and careful curation. We use [conventional commit](https://www.conventionalcommits.org/en/v1.0.0/#summary) titles to help:
* improve communication
* make this process less error-prone and faster

## Commit Message

All ONTAP-MCP commits should follow this form - a GitHub action will reject commits that don't follow this pattern

```
<type>: <description>

[optional body]

[optional footer(s)]
```

`type` is one of:
* `build`
* `chore` - bump version of dependencies
* `ci`
* `doc`
* `feat` - new feature, big or small
* `fix`
* `perf`
* `refactor`
* `revert`
* `style`
* `test`