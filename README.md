# Metisian
metis sequencer explorer.

<img width="1728" alt="스크린샷 2024-09-20 오전 12 35 32" src="https://github.com/user-attachments/assets/7f5047f6-4278-4275-bf00-1d3a691448ce">
<img width="1728" alt="스크린샷 2024-09-20 오전 12 52 35" src="https://github.com/user-attachments/assets/9348bcde-02da-4dab-a6ff-dffa5a7f5798">




Watching for sequencer's status throguh themis RPC, sequencer-set contract and send alerts to defined target(tg, slack, pagerduty, discord).

it checks following.
- tendermint consensus
- l2 commit(also recommit)


```bash
make build

# export VITE_API_HOST="localhost"
make run
```