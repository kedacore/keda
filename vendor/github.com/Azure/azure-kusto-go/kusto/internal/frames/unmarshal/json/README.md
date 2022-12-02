# Why the fork of encoding/json (from version 1.14)

The json package is setup for the general use case. And it makes a lot of assumptions about how people are going to send you data.

In our case, we want to read a stream of Kusto frames represented by a JSON array, but we want to read them as a stream and not all at once.

That means we need to use a json.Decoder. Which is all well and dandy, except what we are going to read out varies by the type of frame. And we won't know what that frame is until we parse the frametype key.

So we decode into a json.RawMessage, extract the frametype from the message, and then unmarshal into the concrete frame.  All good right?

Wrong, because json.RawMessage makes a copy of the bytes that were sent in.  That costs us a huge amount in allocation per frame on 100's of thousands or millions of frames.

Next, for some reason which I can't explain, Go translates numbers in a json.Unmarshal() call into float64 if the target is an interface{}.  I'm sure there is a good reason for it, I just don't understand it.  I would think it would just be an int64 or float64.  Or, because they made it, a json.Number.  The decoder itself can translate to json.Number, if you enable it.  But not Unmarshal.

Because we need to unmarshal to a [][]interface{} so we can translate to a []value.Values(also an interface) that represent rows in a frame, the unmarshal call couldn't hold all numbers we support in a float64.  So Unmarshal was changed to always unmarshal into a json.Number.

# Important changes

- RawMessage now uses the passed slice. This also means you **MUST** decode the raw message into something before making any other decoder calls.
- Unmarshal always unmarshals into a json.Number.

# Things you might try, but won't work

## Doing a RawMessage for the rows instead of [][]interface{}.  Then using another Decoder on that.

You have to store decoders modified for resetting for this not to go crazy in allocs. Also, the decoder doesn't really like being inside content from another decoder, it creates errors about spacing that actually don't cause an error, but it will eat up a lot of allocations (bad design or unintentional consequence).

## Using json-iterator package

Yeah, it looks like it will work, but then it doesn't have the Decoder.Token() thing.

Then you are going to say, "Just remove the first byte to get rid of "[".  

Yeah, been there.  Then it just errors out for some reason that isn't clear.  Also, not a big fan of libraries that say 100% compatibility and then don't.  

I tried a lot of experiments with this library, but never one that worked well.  It just is designed for its use case and not this one.

## Use JSON (insert third party package name)

I looked at a bunch of them.  Don't want to add generated code (its a pain) and others like gojay require weird boiler plate I don't want to bolt on at this time.  

## Use map[string]interface{} and manually convert

Actually, this works fine.  The first iteration did this and was just ever so slightly faster over 5 million records fairly consistently.  Its allocations were signifcantly less that using plain RawMessage for frame discovery and then doing the decode manually into the structs.  

But it was also a lot more code and with a single change to RawMessage I could half the allocations.  On large loads this will keep our GC happy.  Also, just less code.
