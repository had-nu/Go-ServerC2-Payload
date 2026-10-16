# Code of Conduct

## Our purpose

This project is an open-source demonstration of command and control
functionality. It exists for education and for authorised security research.

We welcome contributions. We hold a firm condition: every use must be
authorised.

## Rules for use

### 1. Authorised use only

Use this tool only in a controlled environment where you have explicit
permission.

- Use it on your own systems.
- Use it on systems where the owner gave you written permission.
- Use it in a lab that is designated for security research.

Do not use this tool on any system or network without consent. In many
countries that use is a criminal offence.

### 2. No malicious use

Do not use this project to develop, deploy, or support:

- malware or ransomware
- unauthorised surveillance
- attacks against any person, organisation, or infrastructure

Do not use it to exploit a vulnerability, to interrupt a service, or to cause
damage.

### 3. Transparency

When you use this tool for research or for education:

- state your purpose before you start
- state that the environment is authorised and controlled
- keep a record of the authorisation

Do not share this tool in a way that enables misuse. If you believe that
someone will misuse it, report it. See "Reporting a problem" below.

### 4. The published repository does not contain a working agent

This matters for the people who clone the project.

The repository publishes three programs.

- The **Team Server** accepts check-ins. An operator sends the commands.
- The **Lab Beacon** runs the commands. It needs the shared secret to connect.
- The **Interface Beacon** prints text and stops. It runs no command.

The Lab Beacon is the part that a misuse affects. It needs two things before it
runs anything: a Team Server that an operator started, and the shared secret
from the state directory. A contributor must not remove either check.

Do not add features that give the Lab Beacon persistence, injection, or
stealth beyond the process name. A pull request that adds one of those needs a
reason in the description.

## Reporting a problem

Report a suspected misuse of this tool through GitHub Security Advisories on
the repository page.

For a technical problem, open an issue.

Report privately when a report names a victim, or when a report names a person
who used the tool without authorisation.

## Consequences

We will remove a contribution that breaks these rules. We will close an issue
that supports a malicious use.

We will report illegal activity to the authorities. We will cooperate with a
lawful request.

## Our commitment

We will keep this document current. We will keep the published repository free
of a working agent. We will explain the defensive side of the tool in writing.

## Final note

Build a community that supports ethical security research. Use this tool to
learn. Respect the boundary between a lab and someone else's system.
