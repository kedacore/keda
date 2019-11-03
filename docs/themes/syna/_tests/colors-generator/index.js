const fs = require("fs");
const path = require("path");
const { indexTemplate, index, content, list } = require('./templates');

const root = path.resolve(`${__dirname}/../..`);
const paths = {
  content: path.resolve(`${root}/exampleSite/content`),
  fragments: path.resolve(`${root}/exampleSite/content/fragments`),
  devColors: path.resolve(`${root}/exampleSite/content/dev/colors`)
};

const blacklist = {
  items: ['items-no-content', 'items-only', 'logos-no-content', 'logos-only'],
  embed: ['embed_video'],
  hero: ['header.jpg']
};

const backgrounds = ["white", "light", "secondary", "dark", "primary"];

if (fs.existsSync(paths.devColors)) {
  deleteFolderRecursive(paths.devColors);
}

const fragments = fs.readdirSync(paths.fragments).reduce((tmp, dir) => {
  if (["_index", "_index.md"].indexOf(dir) > -1) {
    return tmp;
  }

  const inside = fs.readdirSync(`${paths.fragments}/${dir}`);
  const fragments = inside
    .filter(
      file =>
        file.match(/\.md$/) &&
        file.indexOf("code-") === -1 &&
        file !== "index.md"
    )
    .reduce((tmp, file) => {
      tmp[file.replace(".md", "")] = `${paths.fragments}/${dir}/${file}`;
      return tmp;
    }, {});

  const nested = inside
    .filter(file =>
      fs.lstatSync(`${paths.fragments}/${dir}/${file}`).isDirectory()
    )
    .reduce((tmp, nDir) => {
      if (["_index", "_index.md"].indexOf(nDir) > -1) {
        return tmp;
      }

      tmp[nDir] = fs
        .readdirSync(`${paths.fragments}/${dir}/${nDir}`)
        .filter(file => file.indexOf("code-") === -1)
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

fs.mkdirSync(paths.devColors);
fs.mkdirSync(`${paths.devColors}/_index`);
fs.writeFile(`${paths.devColors}/_index.md`, index, "utf8", () => {});
fs.writeFile(
  `${paths.devColors}/_index/index.md`,
  content,
  "utf8",
  () => {}
);
fs.writeFile(`${paths.devColors}/_index/list.md`, list, "utf8", () => {});

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

      backgrounds.forEach(background => {
        fs.createReadStream(fragments[fragment].nested[dir][filename]).pipe(
          fs.createWriteStream(
            `${paths.devColors}/${fragment}/${dir}-${background}/${filename}`
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

  if (!content.match(/background\s?=\s".*"/im)) {
    return;
  }

  if (!fs.existsSync(`${paths.devColors}/${fragment}`)) {
    fs.mkdirSync(`${paths.devColors}/${fragment}`);
  }

  if (dir && !fs.existsSync(`${paths.devColors}/${fragment}/${dir}`)) {
    backgrounds.forEach(background => {
      const path = `${paths.devColors}/${fragment}/${dir}-${background}`;
      if (!fs.existsSync(path)) {
        fs.mkdirSync(path);
      }
    });
  }

  fs.writeFile(
    `${paths.devColors}/${fragment}/index.md`,
    indexTemplate.replace(/%fragment%/g, fragment),
    "utf8",
    () => {}
  );

  backgrounds.forEach((background, i) => {
    const tmp = content
      .replace(/background\s?=\s".*"/im, `background = "${background}"`)
      .replace(/weight\s?=\s?"?\d+"?/im, `weight = ${weight + i}`);
    fs.writeFile(
      `${paths.devColors}/${fragment +
        (dir ? `/${dir}-${background}` : "")}/${filename +
        (dir ? "" : `-${background}`)}.md`,
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
