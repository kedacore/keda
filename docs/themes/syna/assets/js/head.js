class Stream {
  constructor() {
    this._topics = {};
    this.subUid = -1;
    this._activeUrlEvent = null;

    this._updateActiveEvent(window.location.href);
    window.onhashchange = function({ newURL }) {
      this._publishHashChange(newURL);
    };

    this.subscribe = this.subscribe.bind(this);
    this.publish = this.publish.bind(this);
    this.unsubscribe = this.unsubscribe.bind(this);
    this._publishHashChange = this._publishHashChange.bind(this);
    this._translateUrlQuery = this._translateUrlQuery.bind(this);
    this._updateActiveEvent = this._updateActiveEvent.bind(this);
  }

  subscribe (topic, func) {
    if (!this._topics[topic]) {
      this._topics[topic] = [];
    }
    const token = (++this.subUid).toString();
    this._topics[topic].push({ token, func });

    if (this._activeUrlEvent && this._activeUrlEvent.event === topic) {
      func.call(null, this._activeUrlEvent.args);
    }
    return token;
  }

  publish(topic, argsText) {
    if (!this._topics[topic]) {
      return false;
    }
    setTimeout(() => {
      const subscribers = this._topics[topic];
      const args = typeof argsText === 'object' ? 
        argsText :
        argsText
          .split(',')
          .reduce((tmp, param) => {
            const [key, value] = param.split(':');
            tmp[key] = value;
            return tmp;
          }, {});

      let len = subscribers ? subscribers.length : 0;
      while (len--) {
        subscribers[len].func.call(null, args);
      }
    }, 0);
    return true;
  }

  unsubscribe(token) {
    for (const topic in this._topics) {
      if (this._topics[topic]) {
        for (let i = 0, j = this._topics[topic].length; i < j; i++) {
          if (this._topics[topic][i].token === token) {
            this._topics[topic].splice(i, 1);
            return token;
          }
        }
      }
    }
    return false;
  }

  _publishHashChange(url) {
    const { event, args } = this._updateActiveEvent(url);
    if (!event) {
      return false;
    }
    return this.publish(event, args);
  }

  _updateActiveEvent(url) {
    let params = this._translateUrlQuery(url);
    let event = null;
    if (!params.e && window.syna.enabledUnsafeEvents && params.event) {
      event = params.event;
    } else if (params.e) {
      params = this._translateUrlQuery(atob(params.e));
      event = params.event;
    } else {
      return {};
    }

    delete params.event;
    this._activeUrlEvent = { event, args: params };
    return this._activeUrlEvent;
  }

  _translateUrlQuery(url) {
    const query = url.slice(url.indexOf('?') + 1) || '';
    return query
      .split('&')
      .reduce((tmp, pair) => {
        const [key, value] = pair.split('=')
        tmp[decodeURIComponent(key)] = decodeURIComponent(value);
        return tmp;
      }, {});
  }
}

class SynaAPI {
  constructor() {
    this._registry = {}
    this.register = this.register.bind(this);
    this.update = this.update.bind(this);
    this.get = this.get.bind(this);
    this.getScope = this.getScope.bind(this);
    this.toArray = this.toArray.bind(this);
  }

  register(scope, id, value) {
    if (!this._registry[scope]) {
      this._registry[scope] = {};
    }

    this._registry[scope][id] = value;
  }

  update(scope, id, value) {
    if (!this._registry[scope] || !this._registry[scope][id]) {
      return null;
    }

    this._registry[scope][id] = value;
    return value;
  }

  get(scope, id) {
    if (!this._registry[scope]) {
      return null;
    }

    return this._registry[scope][id]
  }

  getScope(scope) {
    return this._registry[scope];
  }

  toArray(scope) {
    if (!this._registry[scope]) {
      return null;
    }

    return Object.values(this._registry[scope]);
  }

  renderTemplate(templateString, data) {
    let conditionalMatches, conditionalPattern, copy;
    conditionalPattern = /\$\{\s*isset ([a-zA-Z]*) \s*\}(.*)\$\{\s*end\s*}/g;
    //since loop below depends on re.lastInxdex, we use a copy to capture any manipulations whilst inside the loop
    copy = templateString;
    while (
      (conditionalMatches = conditionalPattern.exec(templateString)) !== null
    ) {
      if (data[conditionalMatches[1]]) {
        //valid key, remove conditionals, leave contents.
        copy = copy.replace(conditionalMatches[0], conditionalMatches[2]);
      } else {
        //not valid, remove entire section
        copy = copy.replace(conditionalMatches[0], '');
      }
    }
    templateString = copy;
    //now any conditionals removed we can do simple substitution
    let key, find, re;
    for (key in data) {
      find = '\\$\\{\\s*' + key + '\\s*\\}';
      re = new RegExp(find, 'g');
      templateString = templateString.replace(re, data[key]);
    }
    return templateString;
  }
}

window.syna = window.syna || {};
window.syna.api = new SynaAPI();
window.syna.stream = new Stream();
window.synaPortals = {};
