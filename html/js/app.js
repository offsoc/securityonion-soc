// Copyright 2019 Jason Ertel (github.com/jertel).
// Copyright 2020-2023 Security Onion Solutions LLC and/or licensed to Security Onion Solutions LLC under one
// or more contributor license agreements. Licensed under the Elastic License 2.0 as shown at
// https://securityonion.net/license; you may not use this file except in compliance with the
// Elastic License 2.0.

const routes = [];

const FEAT_TTR = 'ttr';

const LICENSE_STATUS_ACTIVE = "active";
const LICENSE_STATUS_EXCEEDED = "exceeded";
const LICENSE_STATUS_EXPIRED = "expired";
const LICENSE_STATUS_INVALID = "invalid";
const LICENSE_STATUS_PENDING = "pending";
const LICENSE_STATUS_UNPROVISIONED = "unprovisioned";

const LICENSE_EXPIRES_SOON_DAYS = 45;

const UNREALISTIC_AGE = 1700000000; // About 54 years

const USER_PASSWORD_LENGTH_MIN = 8;
const USER_PASSWORD_LENGTH_MAX = 72;
const USER_PASSWORD_INVALID_RX = /["'$&!]/;

const SYSTEM_USER_ID = '00000000-0000-0000-0000-000000000000';

if (typeof global !== 'undefined') global.routes = routes;

$(document).ready(function() {
  new Vue({
    el: '#app',
    vuetify: new Vuetify({
      icons: {
        iconfont: 'fa',
      },
      theme: {
        dark: true,
        options: {
          customProperties: true,
        },
        themes: {
          light: {
            nav_background: '#12110d',
            nav: '#ffffff',
            drawer_background: '#f4f4f4',
            background: '#ffffff',
          },
          dark: {
            nav_background: '#12110d',
            nav: '#ffffff',
            drawer_background: '#353535',
            background: '#1e1e1e',
          },
        },
      },
    }),
    router: new VueRouter({ routes }),
    data: {
      timestamp: Date.now(),
      i18n: i18n.getLocalizedTranslations(navigator.language),
      loading: false,
      error: false,
      warning: false,
      info: false,
      tip: false,
      errorMessage: "",
      warningMessage: "",
      infoMessage: "",
      tipMessage: "",
      tipTimeout: 6000,
      warningTimeout: 30000,
      errorTimeout: 120000,
      toolbar: null,
      wsUrl: (location.protocol == 'https:' ?  'wss://' : 'ws://') + location.host + location.pathname + 'ws',
      apiUrl: location.origin + location.pathname + 'api/',
      authUrl: '/auth/self-service/',
      settingsUrl: null,
      version: '0.0.0',
      elasticVersion: '0.0.0',
      papi: null,
      connectionTimeout: 300000,
      wsConnectionTimeout: 15000,
      socket: null,
      subscriptions: [],
      parameters: {},
      parametersLoaded: false,
      parameterCallback: null,
      parameterSection: null,
      chartsInitialized: false,
      editorInitialized: false,
      tools: [],
      casesEnabled: false,
      detectionsEnabled: false,
      subtitle: '',
      currentStatus: null,
      connected: false,
      reconnecting: false,
      users: [],
      usersLoadedDate: null,
      cacheRefreshIntervalMs: 300000,
      loadServerSettingsTime: 0,
      user: null,
      username: '',
      maximizedParent: null,
      maximizedOrigWidth: null,
      maximizedOrigHeight: null,
      maximizedCancelFn: null,
      licenseKey: null,
      licenseStatus: null,
      enableReverseLookup: false,
      ip2host: {},
    },
    watch: {
      '$vuetify.theme.dark': 'saveLocalSettings',
      'toolbar': 'saveLocalSettings',
    },
    methods: {
      getMetricsUrl() {
        for (var i = 0; i < this.tools.length; i++) {
          const tool = this.tools[i];
          if (tool.name == "toolInfluxDb") {
            return tool.link;
          }
        }
        return "/";
      },
      target(def) {
        if (localStorage['settings.flags.testing'] === 'true') {
          return "_self";
        }
        return def;
      },
      formatActionContent(content, event, field, value, uriEncode = true) {
        if (!content) return null;

        content = this.replaceActionVar(content, "eventId", event["soc_id"], uriEncode)
        content = this.replaceActionVar(content, "field", field, uriEncode)
        content = this.replaceActionVar(content, "value", value, uriEncode)
        content = this.replaceActionVar(content, "eventJson", JSON.stringify(event))

        const fields = this.getDynamicActionFieldNames(content);
        const route = this;
        if (fields && fields.length > 0) {
          fields.forEach(function(field) {
            value = event[field];
            content = route.replaceActionVar(content, ":" + field, value, uriEncode)
          });
        }
        return content;
      },
      performAction(event, action) {
        if (action && !action.background) return false;
        const options = action.options ? action.options : { mode: 'no-cors' };
        options.method = action.method;
        if (action.method != 'GET') {
          options.body = action.bodyFormatted;
        }
        action.target = localStorage['settings.flags.testing'] !== 'true' ? action.target : '_self';
        const route = this;
        fetch(action.linkFormatted, options)
        .then(data => {
          var link = action.backgroundSuccessLinkFormatted;
          if (link) {
            if (data.status != null) {
              link = route.replaceActionVar(link, "responseCode", data.status, true)
            }
            if (data.statusText != null) {
              link = route.replaceActionVar(link, "responseStatus", data.statusText, true)
            }
            window.open(link, action.target);
          } else {
            route.$root.showTip(route.i18n.actionSuccess + route.$root.localizeMessage(action.name));
          }
        })
        .catch((error) => {
          console.error('Unable to perform background action: ' + error);
          var link = action.backgroundFailureLinkFormatted;
          if (link) {
            link = route.replaceActionVar(link, "error", error.message, true)
            window.open(link, action.target);
          } else {
            route.$root.showTip(route.i18n.actionFailure + route.$root.localizeMessage(action.name));
          }
        });
      },
      base64encode(content) {
        try {
          content = btoa(content);
        } catch (e) {
          console.error("Failed to base64 encode content: " + e);
        }
        return content;
      },
      escape(content) {
        if (content.replace) {
          try {
            content = content.replace(/\\/g, "\\\\");
            content = content.replace(/\"/g, "\\\"");
          } catch (e) {
            console.error("Failed to escape content: " + e);
          }
        }
        return content
      },
      processAncestors(content) {
        content = content.toString();
        if (content.replace) {
          try {
            content = content.replace(/,/g, "\" OR process.entity_id:\"");
          } catch (e) {
            console.error("Failed to set process ancestors for content: " + e);
          }
        }
        return content
      },
      replaceActionVar(content, field, value, uriEncode) {
        if (value === undefined || value == null) return content;

        var encode = function(input) {
          if (uriEncode) {
            return encodeURIComponent(input);
          }
          return input;
        };

        content = content.replace("{" + field + "}", encode(value));
        content = content.replace("{" + field + "|base64}", encode(this.base64encode(value)));
        content = content.replace("{" + field + "|escape}", encode(this.escape(value)));
        content = content.replace("{" + field + "|escape|base64}", encode(this.base64encode(this.escape(value))));
        content = content.replace("{" + field + "|processAncestors}", encode(this.processAncestors(value)));
        return content;
      },
      copyToClipboard(data, style) {
        // Convert entire item into text
        if (style == 'json') {
          data = JSON.stringify(data);
        } else if (style == 'kvp') {
          var text = "";
          for (const prop in data) {
            text += prop + ": " + data[prop] + "\n";
          }
          data = text;
        }

        navigator.clipboard.writeText(data);
      },
      findEligibleActionLinkForEvent(action, event) {
        if (action && action.links) {
          for (var idx = 0; idx < action.links.length; idx++) {
            const link = action.links[idx];

            if (this.isActionLinkEligibleForEvent(link, event)) {
              return link;
            }
          }
        }
        return null;
      },
      isActionLinkEligibleForEvent(link, event) {
        var eligible = true;
        eligible &= (link.indexOf("{eventId}") == -1 || event['soc_id']);
        const fields = this.getDynamicActionFieldNames(link);
        if (fields && fields.length > 0) {
          fields.forEach(function(field) {
            value = event[field];
            eligible &= value != undefined && value != null;
          });
        }
        return eligible;
      },
      getDynamicActionFieldNames(url) {
        const fields = [];
        const matches = url.matchAll(/\{:([@a-zA-Z0-9_.]+?)(\|.*?)?\}/g);
        for (const match of matches) {
          if (match.length > 1) {
            fields.push(match[1]);
          }
        }
        return fields;
      },
      log(msg) {
        console.log(moment().format() + " | " + msg);
      },
      redirectIfAuthCompleted() {
        if (!location.pathname.startsWith("/login")) {
          destUri = this.getCookie("AUTH_REDIRECT");
          if (destUri) {
            this.deleteCookie("AUTH_REDIRECT");
            if (destUri != "/" &&
                !destUri.includes(".?v=") &&
                !destUri.endsWith(".ico") &&
                !destUri.endsWith(".js") &&
                !destUri.endsWith(".css") &&
                !destUri.endsWith(".png") &&
                !destUri.endsWith(".svg") &&
                !destUri.endsWith(".jpg") &&
                !destUri.endsWith(".gif")) {
              this.log("Redirecting to auth destination: " + destUri);
              location.pathname = destUri;
              return true;
            }
          }
        }
        return false;
      },
      redirectRoute() {
        const redirectPage = this.getRedirectPage();
        if (redirectPage) {
          location.hash = '#' + redirectPage;
          this.removeSearchParam('r');
        }
      },
      isLicenseExpiringSoon() {
        if (this.licenseStatus == LICENSE_STATUS_ACTIVE && this.licenseKey.expiration) {
          const now = Date.now();
          const exp = Date.parse(this.licenseKey.expiration);
          const timeToExpirationMs = exp - now;
          const minTimeToExpirationMs = LICENSE_EXPIRES_SOON_DAYS * 24 * 60 * 60 * 1000;
          return timeToExpirationMs < minTimeToExpirationMs;
        }
        return false;
      },
      async loadServerSettings(background) {
        // This version element ensures we're passed the login screen.
        if (document.getElementById("version")) {
          const now = Date.now()
          if (now - this.loadServerSettingsTime > this.cacheRefreshIntervalMs) {
            this.loadServerSettingsTime = now;
            try {
              const response = await this.papi.get('info');
              if (response) {
                this.papi.defaults.headers.common['X-Srv-Token'] = response.data.srvToken;
                this.version = response.data.version;
                this.license = response.data.license;
                this.licenseKey = response.data.licenseKey;
                this.licenseStatus = response.data.licenseStatus;

                if (this.licenseStatus == LICENSE_STATUS_EXCEEDED) {
                  this.showWarning(this.i18n.licenseExceeded);
                } else if (this.licenseStatus == LICENSE_STATUS_PENDING) {
                  this.showWarning(this.i18n.licensePending);
                } else if (this.licenseStatus == LICENSE_STATUS_EXPIRED) {
                  this.showWarning(this.i18n.licenseExpired);
                } else if (this.licenseStatus == LICENSE_STATUS_INVALID) {
                  this.showWarning(this.i18n.licenseInvalid);
                }
                this.parameters = response.data.parameters;
                this.elasticVersion = response.data.elasticVersion;
                this.timezones = response.data.timezones;
                this.enableReverseLookup = response.data.parameters.enableReverseLookup;

                this.user = await this.getUserById(response.data.userId);
                if (this.user) {
                  this.username = this.user.email;
                }

                if (this.parameterCallback != null) {
                  this.parameterCallback(this.parameters[this.parameterSection]);
                  this.parameterCallback = null;
                }
                this.parametersLoaded = true;
                if (this.parameters.webSocketTimeoutMs > 0) {
                  this.wsConnectionTimeout = this.parameters.webSocketTimeoutMs;
                }
                if (this.parameters.apiTimeoutMs > 0) {
                  this.connectionTimeout = this.parameters.apiTimeoutMs;
                }
                if (this.parameters.cacheExpirationMs > 0) {
                  this.cacheRefreshIntervalMs = this.parameters.cacheExpirationMs;
                }
                if (this.parameters.tipTimeoutMs > 0) {
                  this.tipTimeout = this.parameters.tipTimeoutMs;
                }
                if (this.parameters.tools && this.parameters.tools.length > 0) {
                  this.tools = this.parameters.tools;
                  if (this.parameters.inactiveTools) {
                    const inactive = this.parameters.inactiveTools;
                    for (var i = 0; i < this.tools.length; i++) {
                      const tool = this.tools[i];
                      tool.enabled = !inactive.includes(tool.name);
                    }
                  }
                }
                this.casesEnabled = this.parameters.casesEnabled;
                this.detectionsEnabled = this.parameters.detectionsEnabled;

                this.subscribe("status", this.updateStatus);
                this.subscribe('import', (url) => {
                  if (url === 'no-changes') {
                    this.showInfo(this.i18n.gridMemberImportNoChanges);
                  } else if (url) {
                    const u = new URL(url);
                    if (u.host.toUpperCase() == window.location.host.toUpperCase()) {
                      url = u.hash;
                    }
                    const content = this.i18n.gridMemberImportSuccess.replace('{url}', url);
                    this.showInfo(content);
                  }
                });
                this.subscribe('detection-sync', (report) => {
                  const eng = this.correctCasing(report.engine);

                  switch (report.status) {
                    case 'success':
                      this.showInfo(this.i18n.syncSuccess.replace('{engine}', eng));
                      break;
                    case 'partial':
                      this.showWarning(this.i18n.syncPartialSuccess.replace('{engine}', eng));
                      break;
                    case 'error':
                      this.showError(this.i18n.syncFailure.replace('{engine}', eng));
                      break;
                    }
                });
              }
            } catch (error) {
              if (!background) {
                // Only show the error on initial startup, otherwise the error
                // will appear without the user having initiated it and will
                // lead to confusion. There's already a connectivity indicator
                // on the nav bar for the purpose of showing connection state.
                this.showError(error);
              }
            }
          }
        }
      },
      getAuthFlowId() {
        let flow = this.getSearchParam('flow');

        if (flow) {
          localStorage.setItem('flowID', flow);
        } else {
          flow = localStorage.getItem('flowID');
        }

        return flow;
      },
      getRedirectPage() {
        return this.getSearchParam('r');
      },
      getSearchParam(param) {
        const searchParams = new URLSearchParams(window.location.search);
        const value = searchParams.get(param);
        return value;
      },
      removeSearchParam(param) {
        const searchParams = new URLSearchParams(window.location.search);
        searchParams.delete(param);
        window.location.search = searchParams.toString();
      },
      loadParameters(section, callback) {
        if (this.parametersLoaded) {
          callback(this.parameters[section])
        } else {
          this.parameterSection = section;
          this.parameterCallback = callback;
        }
      },
      async logout() {
        try {
          const response = await this.$root.authApi.get('logout/browser');
          location.href = response.data.logout_url;
        } catch (error) {
          this.$root.showError(this.i18n.logoutFailure);
        }
      },
      toggleTheme() {
        this.$vuetify.theme.dark = !this.$vuetify.theme.dark
        this.timestamp = Date.now();
        this.updateEditorTheme();
      },
      setFavicon() {
        const colorSchemeString = window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches
          ? '-dark'
          : '';

        const svgFavicon = document.querySelector('.so-favicon[type="image/svg+xml"]'),
              pngFavicon = document.querySelector('.so-favicon[type="image/png"]');

        if (pngFavicon && svgFavicon) {
          const ext = ".svg";
          var iconIndex = svgFavicon.href.lastIndexOf("/");
          var tagIndex = svgFavicon.href.indexOf("-", iconIndex);
          const extIndex = svgFavicon.href.indexOf(ext, iconIndex);
          if (tagIndex > extIndex) tagIndex = -1;
          const baseText = svgFavicon.href.substring(0, tagIndex !== -1 ? tagIndex : extIndex);
          const queryParam = svgFavicon.href.substring(extIndex + ext.length);

          const attention = this.isAttentionNeeded() ? '-attention' : ''
          pngFavicon.href = `${baseText}${colorSchemeString}${attention}.png${queryParam}`;
          svgFavicon.href = `${baseText}${colorSchemeString}${attention}.svg${queryParam}`;
        }
      },
      setSubtitle(subtitle) {
        this.subtitle = subtitle;
        this.updateTitle();
      },
      updateTitle() {
        var title = "";
        title += this.isAttentionNeeded() ? "! " : "";
        title += "Security Onion";
        if (this.subtitle && this.subtitle.length > 0) {
          title += " - " + this.subtitle;
        }
        document.title = title;
      },
      drawAttention(elementId) {
        var element = $(elementId);
        if (element && element.removeClass) {
          element.removeClass('waggle');
          setTimeout(function() {
            element.addClass('waggle');
          }, 100);
        }
      },
      isLicenseUnprovisioned() {
        return this.licenseStatus == null || this.licenseStatus == LICENSE_STATUS_UNPROVISIONED;
      },
      isLicensed(feat) {
        return this.licenseKey != null && this.licenseStatus == LICENSE_STATUS_ACTIVE &&
            (!this.licenseKey.features.length || this.licenseKey.features.indexOf(feat) != -1);
      },
      colorLicenseStatus(value) {
        if (value == LICENSE_STATUS_ACTIVE) return "success";
        if (value == LICENSE_STATUS_EXCEEDED) return "error";
        if (value == LICENSE_STATUS_EXPIRED) return "warning";
        if (value == LICENSE_STATUS_INVALID) return "error";
        if (value == LICENSE_STATUS_PENDING) return "warning";
        return "info";
      },
      makeHeader(label, value) {
        return { text: label, value: value };
      },
      formatDateTime(date) {
        return this.formatDate(date, this.i18n.dateTimeFormat, this.i18n.dateUnknown);
      },
      formatLocalTimestamp(date, tz) {
        var utcTime = moment.utc(date);
        var localTime = utcTime.tz(tz);
        return localTime.format(this.i18n.timestampFormat);
      },
      formatTimestamp(date) {
        return this.formatDate(date, this.i18n.timestampFormat, this.i18n.dateUnknown);
      },
      formatTimelineLabel(date) {
        return this.formatDate(date, this.i18n.timelineFormat, date);
      },
      formatDate(date, format, dflt) {
        var formatted = dflt;
        if (date) {
          const dateObj = moment(String(date));
          if (dateObj.isAfter('1000-01-01')) {
            formatted = dateObj.format(format);
          }
        }
        return formatted;
      },
      formatDuration(duration) {
        if (duration != null) {
          return moment.duration(duration,"s").humanize();
        }
      },
      formatHours(hours) {
        return this.formatDecimal2(hours);
      },
      formatDecimal1(num) {
        return this.formatDecimalPlaces(num, 1);
      },
      formatDecimal2(num) {
        return this.formatDecimalPlaces(num, 2);
      },
      formatDecimalPlaces(num, places) {
        if (!num) {
          num = 0.0;
        }
        return num.toFixed(places);
      },
      formatCount(count) {
        return Number(count).toLocaleString();
      },
      formatStringArray(strArray) {
        if (strArray != null && strArray.length > 0) {
          return strArray.join(", ");
        }
        return "";
      },
      formatMarkdown(str) {
        marked.setOptions({
          renderer: new marked.Renderer(),
          smartLists: true,
          breaks: true
        })
        marked.use({
          tokenizer: {
            url(src) {
              // Blank function disables bare url tokenization
            }
          }
        })
        var md = str;
        if (str) {
          md = marked.parse(str);
          md = DOMPurify.sanitize(md);
        }
        return md;
      },
      generateDatePickerPreselects() {
        var preselects = {};
        preselects[this.i18n.datePreselectToday] = [moment().startOf('day'), moment().endOf('day')];
        preselects[this.i18n.datePreselectYesterday] = [moment().subtract(1, 'days').startOf('day'), moment().subtract(1, 'days').endOf('day')];
        preselects[this.i18n.datePreselectThisWeek] = [moment().startOf('week'), moment().endOf('week')];
        preselects[this.i18n.datePreselectLastWeek] = [moment().subtract(1, 'week').startOf('week'), moment().subtract(1, 'week').endOf('week')];
        preselects[this.i18n.datePreselectThisMonth] = [moment().startOf('month'), moment().endOf('month')];
        preselects[this.i18n.datePreselectLastMonth] = [moment().subtract(1, 'month').startOf('month'), moment().subtract(1, 'month').endOf('month')];
        preselects[this.i18n.datePreselectPrevious3d] = [moment().subtract(3, 'days').startOf('day'), moment().subtract(1, 'days').endOf('day')];
        preselects[this.i18n.datePreselectPrevious4d] = [moment().subtract(4, 'days').startOf('day'), moment().subtract(1, 'days').endOf('day')];
        preselects[this.i18n.datePreselectPrevious7d] = [moment().subtract(7, 'days').startOf('day'), moment().subtract(1, 'days').endOf('day')];
        preselects[this.i18n.datePreselectPrevious30d] = [moment().subtract(30, 'days').startOf('day'), moment().subtract(1, 'days').endOf('day')];
        preselects[this.i18n.datePreselect3dToNow] = [moment().subtract(3, 'days'), moment()];
        preselects[this.i18n.datePreselect4dToNow] = [moment().subtract(4, 'days'), moment()];
        preselects[this.i18n.datePreselect7dToNow] = [moment().subtract(7, 'days'), moment()];
        preselects[this.i18n.datePreselect30dToNow] = [moment().subtract(30, 'days'), moment()];
        return preselects;
      },
      localizeMessage(origMsg) {
        if (!origMsg) return "";
        var msg = origMsg;
        if (msg.response && msg.response.data) {
          msg = msg.response.data;
          if (msg.error && msg.error.reason) {
            msg = msg.error.reason;
          }
        }
        var localized = this.i18n[msg];
        if (!localized) {
          if (origMsg.message) {
            msg = origMsg.message;
          }
          if (msg.length > 200) {
            msg = msg.substring(0, 200) + "...";
          }
          localized = msg;
        }
        return localized;
      },
      tryLocalize(msg) {
        const localized = this.localizeMessage(msg);
        if (localized) {
          return localized;
        }

        return msg;
      },
      correctCasing(origMsg) {
        const msg = (origMsg+'').toLowerCase();
        var localized = this.i18n['cc_'+msg];
        if (!localized) {
          return origMsg;
        }

        return localized;
      },
      showError(msg) {
        this.error = true;
        this.errorMessage = this.localizeMessage(msg);
        if (this.debug) {
          console.log(msg.stack);
        }
      },
      showWarning(msg) {
        this.warning = true;
        this.warningMessage = this.localizeMessage(msg);
      },
      showInfo(msg) {
        this.info = true;
        this.infoMessage = msg;
      },
      showTip(msg) {
        this.error = false;
        this.warning = false;
        this.info = false;

        this.tip = true;
        this.tipMessage = msg;
      },
      startLoading() {
        this.loading = true;
        this.error = false;
        this.warning = false;
        this.info = false;
      },
      stopLoading() {
        this.loading = false;
      },
      saveLocalSettings() {
        localStorage['settings.app.dark'] = this.$vuetify.theme.dark;
        localStorage['settings.app.navbar'] = this.toolbar;
      },
      loadLocalSettings() {
        if (localStorage['settings.app.dark'] != undefined) {
          this.$vuetify.theme.dark = localStorage['settings.app.dark'] == "true";
          this.updateEditorTheme();
        }
        if (localStorage['settings.app.navbar'] != undefined) {
          this.toolbar = localStorage['settings.app.navbar'] == "true";
        }
      },
      updateEditorTheme() {
        var link = $('link[href^="css/external/prism-custom-"]')[0];
        if (link) {
          if (this.$vuetify.theme.dark) {
            link.href = "css/external/prism-custom-dark-v1.29.0.css";
          } else {
            link.href = "css/external/prism-custom-light-v1.29.0.css";
          }
        }
      },
      subscribe(kind, fn) {
        this.ensureConnected();
        var list = this.subscriptions[kind];
        if (list == undefined) {
          list = [];
          this.subscriptions[kind] = list;
        }
        if (list.indexOf(fn) == -1) {
          list.push(fn);
        }
      },
      unsubscribe(kind, fn) {
        var list = this.subscriptions[kind];
        if (list != undefined) {
          var idx = list.indexOf(fn);
          if (idx > -1) {
            list.splice(idx, 1);
          }
        }
      },
      publish(kind, obj) {
        var listeners = this.subscriptions[kind];
        if (listeners) {
          listeners.forEach(function(listener) {
            listener(obj);
          });
        }
      },
      ensureConnected() {
        if (this.socket == null) {
          this.connected = false;
          this.reconnecting = false;
          this.openWebsocket();
          window.setInterval(this.openWebsocket, this.wsConnectionTimeout);
        }
      },
      openWebsocket() {
        if (!this.socket || this.socket.readyState == WebSocket.CLOSED ) {
          const vm = this;
          this.connected = false;
          this.reconnecting = true;
          this.log("WebSocket connecting to " + this.wsUrl);
          this.socket = new WebSocket(this.wsUrl);
          this.socket.onopen = function(evt) {
            vm.log("WebSocket connected");
            vm.connected = true;
            vm.reconnecting = false;
            vm.updateStatus();
          };
          this.socket.onclose = function(evt) {
            vm.log("WebSocket closed, will attempt to reconnect");
            vm.socket = null;
            vm.connected = false;
            vm.reconnecting = false;
            vm.updateStatus();
          };
          this.socket.onmessage = function(evt) {
            var msg = JSON.parse(evt.data);
            vm.publish(msg.Kind, msg.Object);
          };
          this.socket.onerror = function(evt) {
            vm.log("WebSocket failure: " + evt.data);
          };
        } else {
          this.connected = true;
          this.reconnecting = false;
          try {
            this.socket.send('{ "Kind": "Ping" }');
            this.updateStatus();
          } catch (e) {
            this.log("Failed to ping manager");
            try {
              this.socket.close();
            } catch (ce) {
            }
            this.socket = null;
          }
        }
      },
      showLogin() {
        location.href = this.authUrl + "login/browser";
      },
      checkForUnauthorized(response) {
        if (response) {
          const redirectCookie = this.getCookie('AUTH_REDIRECT');
          if ((response.headers && response.headers['content-type'] == "text/html") ||
              (response.status == 401 && response.request.responseURL.indexOf('/api/') == -1) ||
              (response.request.responseURL.indexOf("/login/banner.md") == -1 && redirectCookie != null && redirectCookie.length > 0)) {
            this.deleteCookie('AUTH_REDIRECT');
            this.showLogin();
            return null
          }
        }
        return response;
      },
      apiSuccessCallback(response) {
        return this.checkForUnauthorized(response);
      },
      apiFailureCallback(error) {
        this.checkForUnauthorized(error.response);
        throw error;
      },
      createApi(baseUrl) {
        const ax = axios.create({
          baseURL: baseUrl,
          timeout: this.connectionTimeout
        });
        ax.interceptors.response.use(this.apiSuccessCallback, this.apiFailureCallback);
        return ax;
      },
      setupApi() {
        this.papi = this.createApi(this.apiUrl);
      },
      setupAuth() {
        this.authApi = axios.create({
          baseURL: this.authUrl,
          timeout: this.connectionTimeout,
          withCredentials: true,
        });
        this.settingsUrl = this.authUrl + 'settings/browser';
      },
      setCookie(name, value, ageSecs) {
        let maxAge = "";
        if (ageSecs) {
          maxAge = ";Max-Age=" + ageSecs;
        }
        document.cookie = name + "=" + value + maxAge + ";Path=/";
      },
      getCookie(name) {
        let cookies = document.cookie.split(';');
        for (let i = 0; i < cookies.length; i++) {
            let cookie = cookies[i].trim();
            let pair = cookie.split("=", 2);
            if (pair.length == 2 && pair[0] == name) {
              return pair[1];
            }
        }
        return null;
      },
      deleteCookie(name) {
        this.setCookie(name, "", -1);
      },
      registerChart(chartType, chartName) {
        var app = this;
        Vue.component(chartName, {
          extends: chartType,
          props: {
            chartdata: { type: Object },
            options: { type: Object }
          },
          mounted () {
            this.renderChart(this.chartdata, this.options)
            this.chartdata.obj = this;
          }
        })
      },
      initializeCharts() {
        if (this.chartsInitialized) return;
        this.registerChart(VueChartJs.Bar, 'bar-chart');
        this.registerChart(VueChartJs.Line, 'line-chart');
        this.registerChart(VueChartJs.Pie, 'pie-chart');

        // Sankey is a separate third-party lib, so use the VueChartJs helper to add the custom chart
        const Sankey = VueChartJs.generateChart('sankey-chart', 'sankey', Chart.controllers['sankey']);
        this.registerChart(Sankey, 'sankey-chart');

        this.chartsInitialized = true;
      },
      registerEditor() {
        Vue.component('prism-editor', PrismEditor.component);
      },
      initializeEditor() {
        if (this.editorInitialized) return;
        this.registerEditor();

        this.editorInitialized = true;
      },
      getColor(colorName, percent = 0) {
        percent = this.$root.$vuetify && this.$root.$vuetify.theme.dark ? percent * -1 : percent;
        var color = colorName;
        if (this.$root.$vuetify && this.$root.$vuetify.theme.currentTheme[colorName]) {
          color = this.$root.$vuetify.theme.currentTheme[colorName];
        }
        var R = parseInt(color.substring(1,3),16);
        var G = parseInt(color.substring(3,5),16);
        var B = parseInt(color.substring(5,7),16);

        R = parseInt(R * (100 + percent) / 100);
        G = parseInt(G * (100 + percent) / 100);
        B = parseInt(B * (100 + percent) / 100);

        R = (R<255)?R:255;
        G = (G<255)?G:255;
        B = (B<255)?B:255;

        var RR = ((R.toString(16).length==1)?"0"+R.toString(16):R.toString(16));
        var GG = ((G.toString(16).length==1)?"0"+G.toString(16):G.toString(16));
        var BB = ((B.toString(16).length==1)?"0"+B.toString(16):B.toString(16));

        return "#"+RR+GG+BB;
      },
      truncate(value, max) {
        const ellipses = "...";
        if (value.length > max + ellipses.length) {
          const half = max / 2;
          value = value.substring(0, half - 1) + ellipses + value.substring(value.length - half);
        }
        return value;
      },
      getAvatar(user) {
        if (user && user.length > 0) {
          return user.charAt(0).toLocaleUpperCase();
        }
        return this.i18n.na;
      },
      async getActiveUsers() {
        const users = await this.getUsers();
        return users.filter(user => user.status != 'locked');
      },
      async getUsers() {
        try {
          const response = await this.papi.get('users/');
          this.users = response.data;
        } catch (error) {
          this.showError(error);
        }
        return this.users;
      },
      async getUserById(id) {
        const nowTime = new Date().time;
        if (this.users.length == 0 || (nowTime - this.usersLoadedTime > this.cacheRefreshIntervalMs)) {
          await this.getUsers();
          this.usersLoadedTime = nowTime;
        }
        return this.getUserByIdViaCache(id);
      },
      getUserByIdViaCache(id) {
        if (this.users) {
          for (var idx = 0; idx < this.users.length; idx++) {
            const user = this.users[idx];
            if (user.id == id) {
              return user;
            }
          }
        }
        return null;
      },
      async populateUserDetails(obj, idField, outputField) {
        if (obj[idField] && obj[idField].length > 0) {
          const id = obj[idField];
          if (id === SYSTEM_USER_ID || id === "agent") {
            Vue.set(obj, outputField, this.i18n.systemUser);
            return
          }

          const user = await this.$root.getUserById(id);
          if (user) {
            Vue.set(obj, outputField, user.email);
          }
        }
      },
      isUserAdmin(user = null) {
        return this.userHasRole("superuser", user);
      },
      isMyUser(user) {
        return user != null && this.user != null && user.id == this.user.id;
      },
      userHasRole(role, user = null) {
        if (!user) {
          user = this.user;
        }

        if (!user) return false;

        return user.roles.indexOf(role) != -1;
      },
      updateStatus(status) {
        if (status) {
          this.currentStatus = status;
        }
        this.setFavicon();
        this.updateTitle();
        this.loadServerSettings(true);
      },
      getDetectionEngines() {
        return ['elastalert', 'strelka', 'suricata'];
      },
      getDetectionEngineStatusClass(engine) {
        switch (this.getDetectionEngineStatus(engine)) {
          case "MigrationFailure": return "warning--text";
          case "SyncFailure": return "warning--text";
          case "IntegrityFailure": return "warning--text";
          case "Healthy": return "success--text";
        }
        return "normal--text";
      },
      getDetectionEngineStatus(engine) {
        if (!this.currentStatus || !this.currentStatus.detections || !this.currentStatus.detections[engine]) {
          return "Unknown";
        }

        const status = this.currentStatus.detections[engine];

        // Order is important in this if/else block. Certain status should take priority. For example,
        // If a sync failure and integrity failure both occurred then show the integrity failure, because
        // if it got to the integrity check then the sync finished but the integrity check failed.
        if (status.migrating) {
          return "Migrating";
        } else if (status.importing && status.syncing) {
          return "Importing";
        } else if (status.migrationFailure) {
          return "MigrationFailure";
        } else if (status.integrityFailure) {
          return "IntegrityFailure";
        } else if (status.syncFailure) {
          return "SyncFailure";
        } else if (status.importing && !status.syncing) {
          return "ImportPending";
        } else if (status.syncing) {
          return "Syncing";
        }
        return "Healthy";
      },
      isDetectionsUnhealthy() {
        return this.currentStatus != null && this.currentStatus.detections != null &&
          ( this.currentStatus.detections.elastalert.integrityFailure ||
            this.currentStatus.detections.suricata.integrityFailure ||
            this.currentStatus.detections.strelka.integrityFailure ||
            this.currentStatus.detections.elastalert.syncFailure ||
            this.currentStatus.detections.suricata.syncFailure ||
            this.currentStatus.detections.strelka.syncFailure ||
            this.currentStatus.detections.elastalert.migrationFailure ||
            this.currentStatus.detections.suricata.migrationFailure ||
            this.currentStatus.detections.strelka.migrationFailure );
      },
      isDetectionsUpdating() {
        return this.currentStatus != null && this.currentStatus.detections != null &&
          !this.isDetectionsUnhealthy() &&
          ( this.currentStatus.detections.elastalert.importing === true ||
            this.currentStatus.detections.elastalert.migrating === true ||
            this.currentStatus.detections.elastalert.syncing === true ||
            this.currentStatus.detections.strelka.importing === true ||
            this.currentStatus.detections.strelka.migrating === true ||
            this.currentStatus.detections.strelka.syncing === true ||
            this.currentStatus.detections.suricata.importing === true ||
            this.currentStatus.detections.suricata.migrating === true ||
            this.currentStatus.detections.suricata.syncing === true );
      },
      isGridUnhealthy() {
        return this.currentStatus && this.currentStatus.grid.unhealthyNodeCount > 0
      },
      isNewAlert() {
        return this.currentStatus && this.currentStatus.alerts.newCount  > 0
      },
      isAttentionNeeded() {
        return this.isNewAlert() || this.isGridUnhealthy() || this.isDetectionsUnhealthy() || !this.connected || this.reconnecting;
      },
      isMaximized() {
        return this.maximizedTarget != null;
      },
      maximizeById(targetId, escapeFn=null) {
        const target = document.getElementById(targetId);
        if (target) {
          return this.maximize(target, escapeFn);
        }
        return false;
      },
      maximize(target, escapeFn=null) {
        this.unmaximize();
        this.maximizedTarget = target;
        this.maximizedOrigWidth = target.style.width;
        this.maximizedOrigHeight = target.style.height;
        target.classList.add("maximized");
        document.documentElement.classList.add("maximized-bg");
        window.scrollTo(0,0);
        this.maximizedCancelFn = escapeFn;
        document.addEventListener('keydown', this.unmaximizeEscapeListener);
        return true;
      },
      unmaximize(userInitiated=false) {
        if (!this.maximizedTarget) return;
        if (userInitiated && this.maximizedCancelFn) {
          if (this.maximizedCancelFn(this.maximizeTarget)) return;
        }
        this.maximizedTarget.classList.remove("maximized");
        document.documentElement.classList.remove("maximized-bg");
        this.maximizedTarget.style.width = this.maximizedOrigWidth;
        this.maximizedTarget.style.height = this.maximizedOrigHeight;
        this.maximizedTarget = null;
        this.maximizedCancelFn = null;
        document.removeEventListener('keydown', this.unmaximizeEscapeListener);
      },
      unmaximizeEscapeListener(event) {
        if (event.code == "Escape") {
          this.unmaximize(true);
        }
        if (typeof event.cancelable !== "boolean" || event.cancelable) {
          event.preventDefault();
        }
      },
      batchLookup(ips, comp) {
        if (!this.enableReverseLookup) {
          return;
        }

        ips = ips.filter(ip => (this.isIPv4(ip) || this.isIPv6(ip)) && !this.ip2host[ip]);
        if (ips.length) {
          ips.forEach(ip => this.ip2host[ip] = []);
          const route = this;
          this.papi.put('util/reverse-lookup', ips).then(response => {
            for (let entry in response.data) {
              let existing = this.ip2host[entry];
              if (!existing) {
                existing = [];
              }

              if (response.data && response.data[entry] && response.data[entry].length) {
                let arr = this.ip2host[entry];
                if (!arr || !arr.length) {
                  arr = [];
                }

                arr.push(...response.data[entry]);
                this.ip2host[entry] = arr;
              }
            }
            comp.$forceUpdate();
          });
        }
      },
      isIPv4(str) {
        if (typeof str === 'string') {
          return !!str.match(/^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/);
        }

        return false;
      },
      isIPv6(str) {
        if (typeof str === 'string') {
          return !!str.match(/^(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))$/i);
        }

        return false;
      },
      pickHostname(ip) {
        if (!this.enableReverseLookup) {
          return '';
        }

        const arr = this.ip2host[ip];
        if (arr && arr.length) {
          const names = this.ip2host[ip].filter(host => host != ip);
          if (names.length) {
            return names[0];
          }
        }

        return '';
      },
      dateAwareSort(items, index, isDesc) {
        items.sort((a, b) => {
          if (index[0] === 'createTime' || index[0] === 'updateTime') {
            if (!isDesc[0]) {
              return new Date(a[index]) - new Date(b[index]);
            }

            return new Date(b[index]) - new Date(a[index]);
          }

          if (typeof a[index] !== 'undefined') {
            if (!isDesc[0]) {
              return a[index].toLowerCase().localeCompare(b[index].toLowerCase());
            }

            return b[index].toLowerCase().localeCompare(a[index].toLowerCase());
          }
        });

        return items;
      },
    },
    created() {
      this.log("Initializing application components");
      if (this.redirectIfAuthCompleted()) return;
      if (this.redirectRoute()) return;
      this.setupApi();
      this.setupAuth();
      this.loadServerSettings(false);
      this.loadLocalSettings();
      Vue.filter('formatDateTime', this.formatDateTime);
      Vue.filter('formatDuration', this.formatDuration);
      Vue.filter('formatHours', this.formatHours);
      Vue.filter('formatDecimal1', this.formatDecimal1);
      Vue.filter('formatDecimal2', this.formatDecimal2);
      Vue.filter('formatCount', this.formatCount);
      Vue.filter('formatMarkdown', this.formatMarkdown);
      Vue.filter('formatTimestamp', this.formatTimestamp);
      $('#app')[0].style.display = "block";
      this.log("Initialization complete");
    },
    mounted() {
      this.setFavicon();
      window.matchMedia('(prefers-color-scheme: dark)').addListener(() => this.setFavicon());
    },
  });
});
