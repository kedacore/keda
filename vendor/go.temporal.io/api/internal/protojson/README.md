_Here lies a vendored copy of protojson with some minor alterations._

Unfortunately we need a few things that the official stack won't let us do:

1. We need to support our old JSON with `camelCase` enums while we migrate to the canonical `SCREAMING_SNAKE_CASE` enums
2. We've decided to support a [shorthand](https://github.com/temporalio/proposals/blob/master/api/http-api.md#payload-formatting) JSON serialization for our Payload type so that our users have a better experience


All code herein is governed by the LICENSE file in this directory (barring the `maybe_marshal` code we added).
