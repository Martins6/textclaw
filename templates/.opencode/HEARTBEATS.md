# Heartbeats

Periodic checks to perform when heartbeat is enabled.

## How Heartbeats Work

1. You will receive a heartbeat trigger with the current state
2. Read this HEARTBEATS.md file for what to check
3. Perform your checks and determine if there's anything worth reporting
4. Use `textclaw notify` only if there's something meaningful

## Always Check

- Check if any files need attention in /workspace/pending/
- Check cronjob status from cronjobs/ directory
- Review recent activity since last heartbeat

## Conditional Checks

- Check {specific topic} if user mentioned wanting updates on it
- Monitor {specific process} if user asked to track it

## Reporting Format

If there's something worth reporting, respond with:
- What changed
- Why it matters
- Any action needed

If nothing worth reporting, respond with exactly: No updates
(This will NOT trigger a notification to the user)
