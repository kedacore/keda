const fs = require("fs");
const path = require("path");
const { indexTemplate, index, content, list } = require('./templates');

const root = path.resolve(`${__dirname}/../..`);
const paths = {
  content: path.resolve(`${root}/exampleSite/content`),
  fragments: path.resolve(`${root}/exampleSite/content/fragments`),
  devAligns: path.resolve(`${root}/exampleSite/content/dev/alignments`)
};

const blacklist = {
  items: ['items-no-content', 'items-only', 'logos-no-content', 'logos-only'],
  embed: ['embed_video'],
  hero: ['header.jpg']
};

const alignments = ["left", "center", "right"];

if (fs.existsSync(paths.devAligns)) {
  deleteFolderRecursive(paths.devAligns);
}

const fragments = fs.readdirSync(paths.fragments).reduce((tmp, dir) => {
  if (["_index", "_index.md"].indexOf(dir) > -1) {
    return tmp;
  }

  const contents = fs.readdirSync(`${paths.fragments}/${dir}`);
  const fragments = contents // Store all fragments that are placed next to their page's index.md
    .filter(
      filename =>
        filename.match(/\.md$/) && // File is .md, meaning it's most likely is a fragment
        filename !== "index.md" && // File isn't the index.md (description)
        filename.indexOf("code-") === -1 && // File isn't a code example
        filename.indexOf("docs") === -1 // File isn't documentation
    )
    .reduce((tmp, file) => {
      tmp[file.replace(".md", "")] = `${paths.fragments}/${dir}/${file}`; // Store the path to the fragment file
      return tmp;
    }, {});

  const nested = contents // Store all fragments that are placed inside a directory next to their page's index.md
    .filter(file =>
      fs.lstatSync(`${paths.fragments}/${dir}/${file}`).isDirectory()
    )
    .reduce((tmp, nDir) => {
      if (["_index", "_index.md"].indexOf(nDir) > -1) {
        return tmp;
      }

      tmp[nDir] = fs
        .readdirSync(`${paths.fragments}/${dir}/${nDir}`)
        .filter(file => file.indexOf("code-") === -1 && file.indexOf("docs") === -1)
        .reduce((tmp, file) => {
          tmp[file] = `${paths.fragments}/${dir}/${nDir}/${file}`;
          return tmp;
        }, {});
      return tmp;
    }, {});

  tmp[dir] = {
    fragments,
    nested
  };

  return tmp;
}, {});

fs.mkdirSync(paths.devAligns);
fs.mkdirSync(`${paths.devAligns}/_index`);
fs.writeFile(`${paths.devAligns}/_index.md`, index, "utf8", () => {});
fs.writeFile(
  `${paths.devAligns}/_index/index.md`,
  content,
  "utf8",
  () => {}
);
fs.writeFile(`${paths.devAligns}/_index/list.md`, list, "utf8", () => {});

Object.keys(fragments).forEach(fragment => {
  let weight = 100;
  Object.keys(fragments[fragment].fragments).forEach(filename => {
    weight += 20;
    parseBlackFriday(
      fragment,
      weight,
      fs.readFileSync(fragments[fragment].fragments[filename], "utf8"),
      filename
    );
  });

  Object.keys(fragments[fragment].nested).forEach(dir => {
    const index = fragments[fragment].nested[dir]["index.md"];
    weight += 20;
    if (parseBlackFriday(fragment, weight, fs.readFileSync(index, "utf8"), "index", dir) === false) {
      return;
    }

    Object.keys(fragments[fragment].nested[dir]).forEach(filename => {
      if (filename === "index.md" || typeof (blacklist[fragment] || []).find(f => f === filename) !== 'undefined') {
        return;
      }

      alignments.forEach(alignment => {
        fs.createReadStream(fragments[fragment].nested[dir][filename]).pipe(
          fs.createWriteStream(
            `${paths.devAligns}/${fragment}/${dir}-${alignment}/${filename}`
          )
        );
      });
    });
  });
});

function parseBlackFriday(fragment, weight, content, filename, dir) {
  if (blacklist[fragment] && typeof blacklist[fragment].find(f => f === filename || f === dir) !== 'undefined') {
    return false;
  }

  if (!content.match(/title\s?=\s".*"/im)) {
    return false;
  }

  if (!fs.existsSync(`${paths.devAligns}/${fragment}`)) {
    fs.mkdirSync(`${paths.devAligns}/${fragment}`);
  }

  if (dir && !fs.existsSync(`${paths.devAligns}/${fragment}/${dir}`)) {
    alignments.forEach(alignment => {
      const path = `${paths.devAligns}/${fragment}/${dir}-${alignment}`;
      if (!fs.existsSync(path)) {
        fs.mkdirSync(path);
      }
    });
  }

  fs.writeFile(
    `${paths.devAligns}/${fragment}/index.md`,
    indexTemplate.replace(/%fragment%/g, fragment),
    "utf8",
    () => {}
  );

  alignments.forEach((alignment, i) => {
    // Edit the fragment configuration
    let tmp = content
      .replace(/^\s*#\s*title_align/im, "title_align")
      .replace(/title_align\s*=\s*"\w+"/im, `title_align = "${alignment}"`)
      .replace(/weight\s?=\s?"?\d+"?/im, `weight = ${weight + i}`);

    if (content.indexOf("title_align") === -1) {
      tmp = tmp.slice(0, tmp.indexOf("+++") + 3) + `\ntitle_align="${alignment}"\n` + tmp.slice(tmp.indexOf("+++") + 3);
    }

    // Write the edited config into the fragment, whether it's in a nested directory or not
    fs.writeFile(
      `${paths.devAligns}/${fragment +
        (dir ? `/${dir}-${alignment}` : "")}/${filename +
        (dir ? "" : `-${alignment}`)}.md`,
      tmp,
      "utf8",
      () => {}
    );
  });
}

function deleteFolderRecursive(path) {
  if (fs.existsSync(path)) {
    fs.readdirSync(path).forEach(file => {
      const curPath = path + "/" + file;
      if (fs.lstatSync(curPath).isDirectory()) {
        deleteFolderRecursive(curPath);
      } else {
        fs.unlinkSync(curPath);
      }
    });
    fs.rmdirSync(path);
  }
};
