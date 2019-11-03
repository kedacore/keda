const indexTemplate = `+++
title = "%fragment%"
fragment = "content"
weight = 100
+++

Different alignments for %fragment% fragment
`;

const index = `+++
title = "Alignments"
+++
`;

const content = `+++
title = "Alignments"
fragment = "content"
weight = 100
headless = true
+++
`;

const list = `+++
fragment = "list"
weight = 110
section = "dev/alignments"
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
