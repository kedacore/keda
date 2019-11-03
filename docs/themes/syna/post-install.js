const fs = require('fs');
const path = require('path');

const mkdir = function(dir) {
  try {
    fs.mkdirSync(dir, 0755);
  } catch(e) {
    if(e.code !== "EEXIST") {
      throw e;
    }
  }
};

const rmdir = function(dir) {
  if (path.existsSync(dir)) {
    const list = fs.readdirSync(dir);
    for(let i = 0; i < list.length; i++) {
      const filename = path.join(dir, list[i]);
      const stat = fs.statSync(filename);

      if(filename === "." || filename === "..") {
        continue
      } else if(stat.isDirectory()) {
        rmdir(filename);
      } else {
        fs.unlinkSync(filename);
      }
    }

    fs.rmdirSync(dir);
  } else {
    console.warn("warn: " + dir + " not exists");
  }
};

const copyDir = function(src, dest) {
  mkdir(dest);
  const files = fs.readdirSync(src);
  for(let i = 0; i < files.length; i++) {
    const current = fs.lstatSync(path.join(src, files[i]));
    if(current.isDirectory()) {
      copyDir(path.join(src, files[i]), path.join(dest, files[i]));
    } else if(current.isSymbolicLink()) {
      const symlink = fs.readlinkSync(path.join(src, files[i]));
      fs.symlinkSync(symlink, path.join(dest, files[i]));
    } else {
      copy(path.join(src, files[i]), path.join(dest, files[i]));
    }
  }
};

const copy = function(src, dest) {
  const oldFile = fs.createReadStream(src);
  const newFile = fs.createWriteStream(dest);
  oldFile.pipe(newFile);
};

copyDir('./node_modules/@fortawesome/fontawesome-free/scss', './assets/styles/fontawesome');
copyDir('./node_modules/@fortawesome/fontawesome-free/webfonts', './static/fonts');
copyDir('./node_modules/bootstrap/scss', './assets/styles/bootstrap');
