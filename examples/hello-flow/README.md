go-cloudthreads example# hello-cloudthreads

## Setup
```
#Vendor Dependencies
glide install -v

fn init hops/hello-cloudthreads

# fn run
fn deploy go-cloudthreads

curl -H "Content-Type: text/plain" -X POST -d "fun" http://localhost:8080/r/go-cloudthreads/hello-cloudthreads
```
