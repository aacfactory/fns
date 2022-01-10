# fns


## 压力测试
K6 结果（50 VUS 30s）

```shell

          /\      |‾‾| /‾‾/   /‾‾/
     /\  /  \     |  |/  /   /  /
    /  \/    \    |     (   /   ‾‾\
   /          \   |  |\  \ |  (‾)  |
  / __________ \  |__| \__\ \_____/ .io

  execution: local
     script: ./test.js
     output: -

  scenarios: (100.00%) 1 scenario, 50 max VUs, 1m0s max duration (incl. graceful stop):
           * default: 50 looping VUs for 30s (gracefulStop: 30s)


running (0m30.0s), 00/50 VUs, 3565697 complete and 0 interrupted iterations
default ✓ [======================================] 50 VUs  30s

     ✓ status was 200

     checks.........................: 100.00% ✓ 3565697       ✗ 0
     data_received..................: 564 MB  19 MB/s
     data_sent......................: 521 MB  17 MB/s
     http_req_blocked...............: avg=1.58µs   min=0s med=0s   max=5.57ms   p(90)=0s      p(95)=0s
     http_req_connecting............: avg=24ns     min=0s med=0s   max=2.31ms   p(90)=0s      p(95)=0s
     http_req_duration..............: avg=261.31µs min=0s med=0s   max=12.92ms  p(90)=844.7µs p(95)=1ms
       { expected_response:true }...: avg=261.31µs min=0s med=0s   max=12.92ms  p(90)=844.7µs p(95)=1ms
     http_req_failed................: 0.00%   ✓ 0             ✗ 3565697
     http_req_receiving.............: avg=26.91µs  min=0s med=0s   max=8.65ms   p(90)=0s      p(95)=29.3µs
     http_req_sending...............: avg=10.53µs  min=0s med=0s   max=7.7ms    p(90)=0s      p(95)=0s
     http_req_tls_handshaking.......: avg=0s       min=0s med=0s   max=0s       p(90)=0s      p(95)=0s
     http_req_waiting...............: avg=223.86µs min=0s med=0s   max=12.43ms  p(90)=641.9µs p(95)=1ms
     http_reqs......................: 3565697 118850.991763/s
     iteration_duration.............: avg=412.23µs min=0s med=92µs max=118.81ms p(90)=1ms     p(95)=1ms
     iterations.....................: 3565697 118850.991763/s
     vus............................: 50      min=50          max=50
     vus_max........................: 50      min=50          max=50

```