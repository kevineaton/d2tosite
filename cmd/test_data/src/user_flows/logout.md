---
title: Flows - Logout
summary:  The flow to lougout of the platform
tags: 
  - users
  - authentication
---

Logging out deletes the `refresh_token` so the user can no longer exchange it for a new `access_token`. Note that since we cache locally at each API ingress, there could be a 5 minute delay between logging out and cache invalidation platform wide!

{{logout_flow}}
