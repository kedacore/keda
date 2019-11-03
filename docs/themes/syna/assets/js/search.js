import Fuse from 'fuse.js';

import $ from './helpers/jq-helpers';

class SynaSearch {
  constructor({ queryParam, searchInput, resultsContainer, template, noResults, empty }) {
    this.searchInput = $(searchInput);
    this.resultsContainer = $(resultsContainer);
    this.template = $(template);
    this.noResults = $(noResults);
    this.empty = $(empty);
    this.fuseOptions = {
      shouldSort: true,
      matchAllTokens: true,
      includeMatches: true,
      tokenize: true,
      threshold: 0.2,
      location: 0,
      distance: 100,
      maxPatternLength: 32,
      minMatchCharLength: 4,
      keys: [
        { name: 'title', weight: 0.8 },
        { name: 'contents', weight: 0.5 },
        { name: 'tags', weight: 0.3 },
        { name: 'categories', weight: 0.3 }
      ]
    };
  
    this.summaryInclude = 60;
    this.indexCache = null;

    const searchQuery = this.param(queryParam) || '';
    this.searchInput.val(searchQuery.trim());
    this.searchInput.on('input', e => this.search(e.target.value.trim()));
    this.search(searchQuery);
  }

  getIndex(callback) {
    if (this.indexCache) {
      return callback(this.indexCache);
    }

    $.ajax({ method: 'get', url: '/index.json' }).then(data => {
      this.indexCache = data;
      callback(data);
    });
  }

  search(query) {
    if (!query) {
      return this.resultsContainer.html(window.syna.api.renderTemplate(this.empty.html(), {}));
    }

    this.getIndex(data => {
      const pages = data;
      const fuse = new Fuse(pages, this.fuseOptions);
      const matches = fuse.search(query);
      if (matches.length > 0) {
        this.populateResults(matches, query);
      } else {
        this.resultsContainer.html(window.syna.api.renderTemplate(this.noResults.html(), {}));
      }
    });
  }

  populateResults(result, query) {
    let finalHTML = '';
    result.forEach((value, key) => {
      const contents = value.contents || value.item.contents;
      if (!contents) return;

      let snippet = '';
      const snippetHighlights = [];
      if (this.fuseOptions.tokenize) {
        snippetHighlights.push(query);
      } else {
        value.matches.forEach(mvalue => {
          if (mvalue.key === 'tags' || mvalue.key === 'categories') {
            snippetHighlights.push(mvalue.value);
          } else if (mvalue.key === 'contents') {
            const start =
              mvalue.indices[0][0] - this.summaryInclude > 0
                ? mvalue.indices[0][0] - this.summaryInclude
                : 0;
            const end =
              mvalue.indices[0][1] + this.summaryInclude < contents.length
                ? mvalue.indices[0][1] + this.summaryInclude
                : contents.length;
            snippet += contents.substring(start, end);
            snippetHighlights.push(
              mvalue.value.substring(
                mvalue.indices[0][0],
                mvalue.indices[0][1] - mvalue.indices[0][0] + 1
              )
            );
          }
        });
      }

      if (snippet.length < 1) {
        snippet += contents.substring(0, this.summaryInclude * 2);
      }
      //pull template from hugo templarte definition
      const templateDefinition = this.template.html();
      //replace values
      let output = window.syna.api.renderTemplate(templateDefinition, {
        key: key,
        title: this.highlight(snippetHighlights, value.item.title),
        link: value.item.permalink,
        tags: value.item.tags,
        categories: value.item.categories,
        snippet: this.highlight(snippetHighlights, snippet)
      });

      finalHTML += output;
    });

    this.resultsContainer.html(finalHTML);
  }

  highlight(highlights, text) {
    return highlights.reduce((tmp, snipvalue) => {
      return tmp.replace(new RegExp(snipvalue, 'im'), `<mark>${snipvalue}</mark>`);
    }, text)
  }

  param(name) {
    return decodeURIComponent(
      (location.search.split(name + '=')[1] || '').split('&')[0]
    ).replace(/\+/g, ' ');
  }
}

window.syna.api.toArray('search').forEach(search => {
  new SynaSearch({
    queryParam: 's',
    searchInput: search.searchInput,
    resultsContainer: search.resultsContainer,
    template: search.template,
    noResults: search.noResults,
    empty: search.empty,
  });
});
