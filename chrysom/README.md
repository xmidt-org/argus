# chrysom
The client library for communicating with Argus over HTTP.
[![PkgGoDev](https://pkg.go.dev/github.com/xmidt-org/argus@v0.5.1/chrysom)](https://pkg.go.dev/github.com/xmidt-org/argus@v0.5.1/chrysom)

## Summary
This package enables CRUD operations with Argus: 

### CRUD Operations

Gets all items that belong to a given owner
```
func (c *Client) GetItems(ctx context.Context, owner string) (Items, error)
```
Updates an item if the item exists and the ownership matches. If the item does not exist then a new item will be created
```
func (c *Client) PushItem(ctx context.Context, owner string, item model.Item) (PushResult, error)
```
Removes the item if it exists and returns the data associated to it.
```
func (c *Client) RemoveItem(ctx context.Context, id, owner string) (model.Item, error)
```

### Listener
The client contains a listener that will listen for item updates from Argus on an interval based on the client configuration. 
``` PullInterval```

Listener provides a mechanism to fetch a copy of all items within a bucket on an interval. If not provided, listening won't be enable for this client.

Start begins listening for updates on an interval given that client configuration is setup correctly. If a listener process is already in progress, calling Start() is a NoOp. If you want to restart the current listener process, call Stop() first.
```
func (c *Client) Start(ctx context.Context) error
```
Stop requests the current listener process to stop and waits for its goroutine to complete. Calling Stop() when a listener is not running (or while one is getting stopped) returns an  error.
```
func (c *Client) Stop(ctx context.Context) error
```