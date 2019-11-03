const indexTemplate = `+++
title = "%fragment%"
fragment = "content"
weight = 100
+++

Different colors for %fragment% fragment
`;

const index = `+++
title = "Colors"
+++
`;

const content = `+++
title = "Colors"
fragment = "content"
weight = 100
headless = true
+++
`;

const list = `+++
fragment = "list"
weight = 110
section = "dev/colors"
count = 1000
summary = false
tiled = true
subsections = false
+++
`;

module.exports = {
  indexTemplate,
  index,
  content,
  list,
}
