+++
fragment = "content"
weight = 111
+++

<details><summary>Code</summary>
```+++
fragment = "toc"
weight = 110
background = "secondary"
content = "content.md"
+++

```
</details>

<details><summary>Code (content.md)</summary>
```+++
fragment = "content"
weight = 111
disabled = true # This is just to prevent rendering of the content on the documentation
+++

# Sample header 1
## Sample header 2
### Sample header 3
## Sample header 2

```
</details>
