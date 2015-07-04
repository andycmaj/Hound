// TODO(knorton): Use something to bundle this more intelligently and get this
// out of the global scope.

var lib = {

    ExpandVars: function(template, values) {
      for (var name in values) {
        template = template.replace('{' + name + '}', values[name]);
      }
      return template;
    },

    UrlToRepo: function(repo, path, line, rev) {
        var url = repo.url.replace(/\.git$/, ''),
            pattern = repo['url-pattern'],
            anchor = line ? lib.ExpandVars(pattern.anchor, { line : line }) : '';

        // Hacky solution to fix _some more_ of the 404's when using SSH style URLs.
        // This works for both github style URLs (git@github.com:username/Foo.git) and
        // bitbucket style URLs (ssh://hg@bitbucket.org/username/Foo).

        // Regex explained: Match either `git` or `hg` followed by an `@`.
        // Next, slurp up the hostname by reading until either a `:` or `/` is found.
        // Finally, grab all remaining characters.
        var sshParts = /(git|hg)@(.*?)(:|\/)(.*)/.exec(url);
        if (sshParts) {
          url = '//' + sshParts[2] + '/' + sshParts[4];
        }

        // I'm sure there is a nicer React/jsx way to do this:
        return lib.ExpandVars(pattern['base-url'], {
          url : url,
          path: path,
          rev: rev,
          anchor: anchor
        });
    }
};

var Signal = function() {
};

Signal.prototype = {
  listeners : [],

  tap: function(l) {
    // Make a copy of the listeners to avoid the all too common
    // subscribe-during-dispatch problem
    this.listeners = this.listeners.slice(0);
    this.listeners.push(l);
  },

  untap: function(l) {
    var ix = this.listeners.indexOf(l);
    if (ix == -1) {
      return;
    }

    // Make a copy of the listeners to avoid the all to common
    // unsubscribe-during-dispatch problem
    this.listeners = this.listeners.slice(0);
    this.listeners.splice(ix, 1);
  },

  raise: function() {
    var args = Array.prototype.slice.call(arguments, 0);
    this.listeners.forEach(function(l) {
      l.apply(this, args);
    });
  }
};

var css = function(el, n, v) {
  el.style.setProperty(n, v, '');
};

var FormatNumber = function(t) {
  var s = '' + (t|0),
      b = [];
  while (s.length > 0) {
    b.unshift(s.substring(s.length - 3, s.length));
    s = s.substring(0, s.length - 3);
  }
  return b.join(',');
};

var ParamsFromQueryString = function(qs, params) {
  params = params || {};

  if (!qs) {
    return params;
  }

  qs.substring(1).split('&').forEach(function(v) {
    var pair = v.split('=');
    if (pair.length != 2) {
      return;
    }
    if (pair[1].indexOf(',') >= 0) {
      params[decodeURIComponent(pair[0])] = pair[1].split(',');
    } else {
      params[decodeURIComponent(pair[0])] = decodeURIComponent(pair[1]);
    }
  });

  if (params["repos"] === '') {
    params["repos"] = '*';
  }

  return params;
};

var ParamsFromUrl = function(params) {
  params = params || {
    q: '',
    i: 'nope',
    files: '',
    repos: '*'
  };
  return ParamsFromQueryString(location.search, params);
};

var ParamValueToBool = function(v) {
  v = v.toLowerCase();
  return v == 'fosho' || v == 'true' || v == '1';
};

var css = function(el, n, v) {
  el.style.setProperty(n, v, '');
};

var FormatNumber = function(t) {
  var s = '' + (t|0),
      b = [];
  while (s.length > 0) {
    b.unshift(s.substring(s.length - 3, s.length));
    s = s.substring(0, s.length - 3);
  }
  return b.join(',');
};

var ParamsFromQueryString = function(qs, params) {
  params = params || {};

  if (!qs) {
    return params;
  }

  qs.substring(1).split('&').forEach(function(v) {
    var pair = v.split('=');
    if (pair.length != 2) {
      return;
    }
    if (pair[1].indexOf(',') >= 0) {
      params[decodeURIComponent(pair[0])] = pair[1].split(',');
    } else {
      params[decodeURIComponent(pair[0])] = decodeURIComponent(pair[1]);
    }
  });

  if (params["repos"] === '') {
    params["repos"] = '*';
  }

  return params;
};

var ParamsFromUrl = function(params) {
  params = params || {
    q: '',
    i: 'nope',
    files: '',
    repos: '*'
  };
  return ParamsFromQueryString(location.search, params);
};

var ParamValueToBool = function(v) {
  v = v.toLowerCase();
  return v == 'fosho' || v == 'true' || v == '1';
};


var css = function(el, n, v) {
  el.style.setProperty(n, v, '');
};

var FormatNumber = function(t) {
  var s = '' + (t|0),
      b = [];
  while (s.length > 0) {
    b.unshift(s.substring(s.length - 3, s.length));
    s = s.substring(0, s.length - 3);
  }
  return b.join(',');
};

var ParamsFromQueryString = function(qs, params) {
  params = params || {};

  if (!qs) {
    return params;
  }

  qs.substring(1).split('&').forEach(function(v) {
    var pair = v.split('=');
    if (pair.length != 2) {
      return;
    }
    if (pair[1].indexOf(',') >= 0) {
      params[decodeURIComponent(pair[0])] = pair[1].split(',');
    } else {
      params[decodeURIComponent(pair[0])] = decodeURIComponent(pair[1]);
    }
  });

  if (params["repos"] === '') {
    params["repos"] = '*';
  }

  return params;
};

var ParamsFromUrl = function(params) {
  params = params || {
    q: '',
    i: 'nope',
    files: '',
    repos: '*'
  };
  return ParamsFromQueryString(location.search, params);
};

var ParamValueToBool = function(v) {
  v = v.toLowerCase();
  return v == 'fosho' || v == 'true' || v == '1';
};


var css = function(el, n, v) {
  el.style.setProperty(n, v, '');
};

var FormatNumber = function(t) {
  var s = '' + (t|0),
      b = [];
  while (s.length > 0) {
    b.unshift(s.substring(s.length - 3, s.length));
    s = s.substring(0, s.length - 3);
  }
  return b.join(',');
};

var ParamsFromQueryString = function(qs, params) {
  params = params || {};

  if (!qs) {
    return params;
  }

  qs.substring(1).split('&').forEach(function(v) {
    var pair = v.split('=');
    if (pair.length != 2) {
      return;
    }
    if (pair[1].indexOf(',') >= 0) {
      params[decodeURIComponent(pair[0])] = pair[1].split(',');
    } else {
      params[decodeURIComponent(pair[0])] = decodeURIComponent(pair[1]);
    }
  });

  if (params["repos"] === '') {
    params["repos"] = '*';
  }

  return params;
};

var ParamsFromUrl = function(params) {
  params = params || {
    q: '',
    i: 'nope',
    files: '',
    repos: '*'
  };
  return ParamsFromQueryString(location.search, params);
};

var ParamValueToBool = function(v) {
  v = v.toLowerCase();
  return v == 'fosho' || v == 'true' || v == '1';
};


