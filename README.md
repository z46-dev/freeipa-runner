# freeipa-runner

A lightweight application allowing for distribution and execution of ansible playbooks and other simple scripts

## The Problem

On a FreeIPA domain, you may have a number of Linux clients that need to be configured in a similar way. You could use Ansible to manage these clients, but that requires setting up an Ansible control node and managing SSH keys. Alternatively, you could use FreeIPA's built-in configuration management features, but those can be complex and difficult to set up.

This application hooks into FreeIPA's existing authentication and authorization mechanisms, allowing you to easily distribute and execute scripts on your clients without the need for a separate control node or complex configuration management setup.

## Usage

### Installing the daemon

### Manual Running

### Scheduled Running