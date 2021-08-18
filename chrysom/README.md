# chrysom
The client library for communicating with Argus over HTTP.

[![Go Reference](https://pkg.go.dev/badge/github.com/xmidt-org/argus/chrysom.svg)](https://pkg.go.dev/github.com/xmidt-org/argus/chrysom)

## Summary
This package enables CRUD operations on the items stored in Argus.  The items being stored are valid JSON objects. The package also provides a listener that is able to poll from Argus on an interval.

## CRUD Operations

- On a `GetItems()` call, chrysom returns a list of all items belonging to the provided owner.
- `PushItem()` updates an item if the item exists and the ownership matches. If the item does not exist then a new item will be created. It will return an error or a string saying whether the item was successfully created or updated.
- `RemoveItem()` removes the item if it exists and returns the data associated to it.

## Listener
The client contains a listener that will listen for item updates from Argus on an interval based on the client configuration. 

Listener provides a mechanism to fetch a copy of all items within a bucket on an interval. If not provided, listening won't be enable for this client.

- Start begins listening for updates on an interval. A Listener must be given in the ListenerConfig for this to work. If a listener process is already in progress, calling Start() is a NoOp. If you want to restart the current listener process, call Stop() first.
- Stop requests the current listener process to stop and waits for its goroutine to complete. Calling Stop() when a listener is not running (or while one is getting stopped) returns an  error.