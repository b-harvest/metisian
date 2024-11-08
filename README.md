# Metisian
metis sequencer explorer.

<img width="698" alt="screenshot 2024-11-09 am 12 41 56" src="https://github.com/user-attachments/assets/a48201ab-c8db-43a7-ada9-82fb18cf8fc5">





Watching for sequencer's status throguh themis RPC, sequencer-set contract and send alerts to defined target(tg, slack, pagerduty, discord).

it checks following.
- tendermint consensus
- l2 commit(also recommit)


### dashboard
```bash
git clone https://github.com/b-harvest/metisian

cd dashboard

make build
# export VITE_API_HOST="localhost"
make run
```
