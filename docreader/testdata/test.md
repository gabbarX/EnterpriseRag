# Test Markdown Document

This is a test Markdown document, used to exercise the Markdown parsing pipeline.

## With an image

![Test image](https://geektutu.com/post/quick-go-protobuf/go-protobuf.jpg)

## With a link

This is a [test link](https://example.com).

## With a code block

```python
def hello_world():
    print("Hello, World!")
```

## With a table

| Leave Type | Days |
|-------|-------|
| Earned Leave | 18 |
| Casual Leave | 12 |

## Chunking test

This section exercises chunking, so that the Markdown structure stays intact when it is split.

- First block of content
- Second block of content
- Third block of content

## Overlap test

This section may overlap with the blocks before and after it, so that the context stays continuous. 