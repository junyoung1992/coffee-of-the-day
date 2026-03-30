# Fly.io Deployment Guide

## Goal
Deploy coffee-of-the-day to Fly.io with:
- single app
- region nrt
- SQLite at /data/coffee.db
- persistent volume
- health check at /health
- JWT secret
- later GitHub Actions auto deploy

## Preconditions
- Dockerizing is done on feat/1-fly-deployment
- backend must listen on 0.0.0.0
- internal port must match fly.toml
- /health must return 200

## Fly.io Website Tasks
- sign in to Fly.io
- register billing info
- after first deploy, verify app/machine/volume/logs in dashboard
- later add FLY_API_TOKEN to GitHub Actions secrets

## CLI Tasks
1. fly auth login
2. fly launch --no-deploy
3. edit fly.toml
4. create volume
5. set JWT_SECRET
6. fly deploy
7. verify /health and persistence

## fly.toml Requirements
- primary_region = "nrt"
- mount /data
- env DB_PATH=/data/coffee.db
- health check path /health
- internal_port must match app port

## Suggested Commands
```bash
fly auth login
fly launch --no-deploy
fly volumes create coffee_data --region nrt --size 1
fly secrets set JWT_SECRET=...
fly deploy
fly status
fly logs
```

## Notes
-	Prefer first deploy without Litestream if debugging needs to be simpler
-	Add GitHub Actions after manual deploy is confirmed

When working on deployment:
- keep infrastructure minimal and low-cost
- prefer Fly.io single app + single volume
- region must be nrt
- DB path must be /data/coffee.db
- health check path must be /health
- do not introduce paid infrastructure unless explicitly approved

## Definition of Done

Deployment is considered successful only if:

- fly deploy succeeds without errors
- /health endpoint returns 200
- application is accessible via Fly URL
- data written to SQLite persists after redeploy
- Fly dashboard shows:
  - 1 running machine in nrt region
  - volume attached correctly
  - no crash/restart loop

## Common Pitfalls

- app listens on localhost instead of 0.0.0.0
- internal_port mismatch with actual server port
  - app must listen on 0.0.0.0:8080 (or PORT env)
  - fly.toml internal_port must match this port
- /health endpoint does not return 200
- volume name mismatch between fly.toml and created volume
- DB path not pointing to /data/coffee.db
- SQLite file created in non-persistent path

## Instructions for Claude Code

Before making any changes:

1. Summarize this document
2. Identify missing implementation in the codebase
3. Propose a step-by-step plan

While working:
- Apply changes incrementally
- Explain each change briefly
- Do not introduce additional infrastructure
- Keep cost minimal

After deployment:
- Verify all Definition of Done criteria
- If any check fails, fix before proceeding
