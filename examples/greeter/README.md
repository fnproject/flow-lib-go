Companion function example to be used as invoke target from hello-flow example.

To resolve the function ID to use in `flows.CurrentFlow().InvokeFunction`:

```
make deploy
fn inspect function greeter greeter | grep fnproject.io/fn/invokeEndpoint
```
