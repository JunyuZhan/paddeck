(function () {
  "use strict";

  /* ================= palette (theme driven) ================= */

  var THEMES = window.THEMES || [];
  var currentTheme = null;
  var C = {};            // filled by setTheme()
  var ACC = {};

  function setTheme(t) {
    if (!t) return;
    currentTheme = t;
    C = {
      bg: t.bg, text: t.text, dim: t.sub, faint: t.ov0, neutral: t.sub1,
      mauve: t.a1, teal: t.a2, pink: t.a3, lavender: t.a4, peach: t.a5, sky: t.a6,
      sapphire: t.a6, green: t.ok, yellow: t.warn, red: t.crit,
      off: t.ov0, line: t.border
    };
    ACC = { c: t.a1, t: t.a2, v: t.a3, p: t.a4, s: t.a5, m: t.a6 };
  }
  setTheme(THEMES[0] || {
    bg: "#1e1e2e", panel: "#313244", border: "#45475a", text: "#cdd6f4",
    sub: "#a6adc8", sub1: "#bac2de", ov0: "#6c7086",
    a1: "#cba6f7", a2: "#94e2d5", a3: "#f5c2e7", a4: "#b4befe", a5: "#fab387", a6: "#89dceb",
    ok: "#a6e3a1", warn: "#f9e2af", crit: "#f38ba8"
  });

  var BGS = [
    { id: "aurora", name: "极光 AURORA", cls: "fx-aurora" },
    { id: "mesh", name: "弥散 MESH", cls: "fx-mesh" },
    { id: "dots", name: "点阵 DOT", cls: "fx-dots" },
    { id: "retro", name: "网格 RETRO", cls: "fx-retro" },
    { id: "none", name: "纯色 NONE", cls: "fx-none" }
  ];

  var hostMap = {};

  var PAGES = [
    { id: "overview",   label: "OVERVIEW",   acc: "c" },
    { id: "hosts",      label: "HOSTS",      acc: "t" },
    { id: "containers", label: "CONTAINERS", acc: "v" },
    { id: "pve",        label: "VIRTUAL",    acc: "p" },
    { id: "services",   label: "SERVICES",   acc: "s" },
    { id: "trend",      label: "TREND",      acc: "m" }
  ];

  var REFRESH_MS = 5000;
  var PAGE_MS = 20000;
  var TREND_MS = 60000;

  /* ================= icons (Lucide-style, 24x24) ================= */

  var ICONS = {
    server: '<rect x="2" y="3" width="20" height="7" rx="1.5"/><rect x="2" y="14" width="20" height="7" rx="1.5"/><path d="M6 6.5h.01M6 17.5h.01"/>',
    box: '<path d="M21 8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16Z"/><path d="m3.3 7 8.7 5 8.7-5"/><path d="M12 22V12"/>',
    globe: '<circle cx="12" cy="12" r="9.5"/><path d="M2.5 12h19"/><path d="M12 2.5a15 15 0 0 1 4 9.5 15 15 0 0 1-4 9.5 15 15 0 0 1-4-9.5 15 15 0 0 1 4-9.5Z"/>',
    monitor: '<rect x="2" y="3" width="20" height="14" rx="2"/><path d="M8 21h8M12 17v4"/>',
    layers: '<path d="m12.8 2.2a2 2 0 0 0-1.6 0L2.6 6.1a1 1 0 0 0 0 1.8l8.6 3.9a2 2 0 0 0 1.6 0l8.6-3.9a1 1 0 0 0 0-1.8Z"/><path d="m22 17.7-9.2 4.1a2 2 0 0 1-1.6 0L2 17.7"/><path d="m22 12.7-9.2 4.1a2 2 0 0 1-1.6 0L2 12.7"/>',
    database: '<ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v14a9 3 0 0 0 18 0V5"/><path d="M3 12a9 3 0 0 0 18 0"/>',
    cloud: '<path d="M17.5 19a4.5 4.5 0 0 0 0-9 6 6 0 0 0-11.6 1.6A3.5 3.5 0 0 0 6.5 19Z"/>',
    book: '<path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2Z"/>',
    music: '<path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/>',
    git: '<circle cx="18" cy="18" r="3"/><circle cx="6" cy="6" r="3"/><path d="M6 21V9a9 9 0 0 0 9 9"/>',
    file: '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8Z"/><path d="M14 2v6h6"/>',
    cpu: '<rect x="4" y="4" width="16" height="16" rx="2"/><rect x="9" y="9" width="6" height="6"/><path d="M9 2v2M15 2v2M9 20v2M15 20v2M2 9h2M2 15h2M20 9h2M20 15h2"/>',
    hdd: '<path d="M22 12H2"/><path d="M5.5 5.1 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.5-6.9A2 2 0 0 0 16.8 4H7.2a2 2 0 0 0-1.7 1.1Z"/><path d="M6 16h.01M10 16h.01"/>',
    wifi: '<path d="M5 12.6a11 11 0 0 1 14 0"/><path d="M1.5 9a16 16 0 0 1 21 0"/><path d="M8.5 16.1a6 6 0 0 1 7 0"/><path d="M12 20h.01"/>',
    router: '<rect x="2" y="13" width="20" height="8" rx="2"/><path d="M6.5 17h.01M10 17h.01"/><path d="M12 13V9"/><path d="M8.2 9a5.5 5.5 0 0 1 7.6 0"/>',
    activity: '<path d="M22 12h-4l-3 9L9 3l-3 9H2"/>',
    shield: '<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10Z"/>',
    image: '<rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><path d="m21 15-5-5L5 21"/>',
    home: '<path d="m3 10 9-7 9 7v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z"/><path d="M9 22V12h6v10"/>',
    download: '<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><path d="m7 10 5 5 5-5"/><path d="M12 15V3"/>',
    lock: '<rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>',
    mail: '<rect x="2" y="4" width="20" height="16" rx="2"/><path d="m22 7-10 6L2 7"/>',
    camera: '<path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3Z"/><circle cx="12" cy="13" r="3"/>',
    search: '<circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/>',
    zap: '<path d="M13 2 3 14h8l-1 8 10-12h-8Z"/>',
    terminal: '<path d="m4 17 6-6-6-6"/><path d="M12 19h8"/>',
    film: '<rect x="2" y="2" width="20" height="20" rx="2"/><path d="M7 2v20M17 2v20M2 12h20M2 7h5M2 17h5M17 17h5M17 7h5"/>'
  };

  var RULES = [
    [/postgres|mysql|maria|mongo|redis|influx|clickhouse|minio|database|\bdb\b/, "database"],
    [/plex|jellyfin|emby|navidrome|sonic|music|audio|spotify|airsonic/, "music"],
    [/immich|photo|gallery|picture|image|\bpic\b|camera|frigate|nvr/, "image"],
    [/git|gitea|forgejo|github|gitlab/, "git"],
    [/home|hass|homeassistant|zigbee|automation/, "home"],
    [/nginx|caddy|traefik|haproxy|proxy|web|apache|http/, "globe"],
    [/router|openwrt|wifi|network|dhcp|dns|adguard|pihole|gateway|ikuai/, "router"],
    [/nas|synology|dsm|storage|backup|s3|archive/, "hdd"],
    [/grafana|prometheus|uptime|status|monitor|alert|victoria/, "activity"],
    [/qbittorrent|transmission|deluge|aria|download|sabnzbd|nzb/, "download"],
    [/vault|auth|keycloak|secret|password|authelia/, "lock"],
    [/mail|smtp|imap|postfix|roundcube/, "mail"],
    [/wiki|docs|book|notion|outline/, "book"],
    [/search|elastic|meili|typesense|sonar/, "search"],
    [/plex|stream|video|media|film/, "film"],
    [/docker|container|package|app|api|service/, "box"]
  ];

  /* category accent: keeps a 30-tile grid from reading as one flat colour */
  var CAT_ACC = {
    database: "t", music: "v", image: "m", git: "v", home: "s", globe: "c",
    router: "s", hdd: "v", activity: "c", download: "t", lock: "p", mail: "m",
    book: "v", search: "c", film: "m", box: "c"
  };

  function catAccent(text) {
    var t = String(text || "").toLowerCase();
    for (var i = 0; i < RULES.length; i++) {
      if (RULES[i][0].test(t)) return CAT_ACC[RULES[i][1]] || "c";
    }
    return "c";
  }

  /* ================= state ================= */

  var pageIndex = 0;
  var snapshot = null;
  var hist = {};
  var lastTrend = 0;
  var pauseUntil = 0;
  var W = 0;

  /* ================= helpers ================= */

  function $(id) { return document.getElementById(id); }

  function esc(s) {
    if (s === null || s === undefined) return "";
    return String(s).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
  }

  function clamp(v) {
    if (!v || isNaN(v)) return 0;
    return v < 0 ? 0 : (v > 100 ? 100 : v);
  }

  function pct(used, total) {
    if (!total || total <= 0) return 0;
    return used / total * 100;
  }

  function fmtBytes(n) {
    if (!n || n <= 0) return "--";
    var units = ["B", "K", "M", "G", "T"];
    var i = 0, v = n;
    while (v >= 1024 && i < units.length - 1) { v = v / 1024; i++; }
    return (v >= 100 ? Math.round(v) : v.toFixed(1)) + units[i];
  }

  function fmtRate(n) {
    if (!n || n <= 0) return "0";
    return fmtBytes(n) + "/s";
  }

  function fmtPct(v) {
    if (v === null || v === undefined || isNaN(v)) return "--";
    return (v >= 10 ? Math.round(v) : v.toFixed(1)) + "%";
  }

  function fmtUptime(sec) {
    if (!sec) return "--";
    var d = Math.floor(sec / 86400);
    var h = Math.floor((sec % 86400) / 3600);
    if (d >= 100) return d + "d";
    if (d > 0) return d + "d" + h + "h";
    if (h > 0) return h + "h";
    return Math.floor((sec % 3600) / 60) + "m";
  }

  function xhr(url, cb) {
    var req = new XMLHttpRequest();
    req.open("GET", url, true);
    req.timeout = 15000;
    req.onreadystatechange = function () {
      if (req.readyState !== 4) return;
      if (req.status >= 200 && req.status < 300) {
        try { cb(null, JSON.parse(req.responseText)); }
        catch (e) { cb(e, null); }
      } else {
        cb(new Error("HTTP " + req.status), null);
      }
    };
    req.ontimeout = function () { cb(new Error("timeout"), null); };
    req.onerror = function () { cb(new Error("network"), null); };
    req.send();
  }

  /* cached refs inside a tile (structure is stable between rebuilds) */

  function refs(el) {
    if (el._r) return el._r;
    var r = el._r = {};
    r.icon = el.querySelector(".ico svg");
    r.marker = el.querySelector(".mk");
    r.name = el.querySelector(".t-name, .sm-name, .hcard-name");
    r.sub = el.querySelector(".t-sub, .sm-sub, .hcard-sub");
    r.netD = el.querySelector(".dn");
    r.netU = el.querySelector(".up");
    r.bars = el.querySelectorAll(".bl-b i");
    r.barV = el.querySelectorAll(".bl-v");
    r.wanD = el.querySelector(".wn-d");
    r.wanU = el.querySelector(".wn-u");
    r.statsV = el.querySelectorAll(".stat-v");
    r.statsK = el.querySelectorAll(".stat-k");
    r.spark = el.querySelector(".sp-l");
    r.sparkA = el.querySelector(".sp-a");
    r.sparkT = el.querySelector(".sp-t");
    r.smBar = el.querySelector(".sm-bar i");
    r.cstat = el.querySelector(".c-stat");
    r.chost = el.querySelector(".c-host");
    r.svcV = el.querySelector(".svc-v");
    r.svcWord = el.querySelector(".svc-word");
    r.svcMeta = el.querySelector(".svc-meta");
    return r;
  }

  /* ================= icons / markers ================= */

  function icon(name, size, color) {
    var body = ICONS[name] || ICONS.box;
    return '<svg viewBox="0 0 24 24" width="' + size + '" height="' + size + '" fill="none" stroke="' +
      color + '" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">' + body + "</svg>";
  }

  function iconFor(text, size, color, fallback) {
    var t = String(text || "").toLowerCase();
    for (var i = 0; i < RULES.length; i++) {
      if (RULES[i][0].test(t)) return icon(RULES[i][1], size, color);
    }
    return icon(fallback || "box", size, color);
  }

  function marker(state, size) {
    var s = size || 12;
    if (state === "crit") {
      return '<svg width="' + s + '" height="' + s + '" viewBox="0 0 12 12"><path d="M3 3L9 9M9 3L3 9" stroke="' + C.red + '" stroke-width="2" stroke-linecap="round"/></svg>';
    }
    if (state === "warn") {
      return '<svg width="' + s + '" height="' + s + '" viewBox="0 0 12 12"><path d="M6 1.2L11.2 10.6H0.8Z" fill="' + C.yellow + '"/></svg>';
    }
    if (state === "off") {
      return '<svg width="' + s + '" height="' + s + '" viewBox="0 0 12 12"><circle cx="6" cy="6" r="3.6" fill="none" stroke="' + C.off + '" stroke-width="1.6"/></svg>';
    }
    return '<svg width="' + s + '" height="' + s + '" viewBox="0 0 12 12"><circle cx="6" cy="6" r="3.6" fill="' + C.green + '"/></svg>';
  }

  function iconColor(state, acc) {
    if (state === "crit") return C.red;
    if (state === "warn") return C.yellow;
    if (state === "off") return C.off;
    return acc ? (ACC[acc] || C.neutral) : C.neutral;
  }

  function hostIcon(h) {
    if (h.kind === "ikuai") return "router";
    var t = ((h.id || "") + " " + (h.name || "") + " " + ((h.labels || []).join(" "))).toLowerCase();
    if (/proxmox|\bpve\b/.test(t)) return "layers";
    if (/nas|synology|dsm|storage/.test(t)) return "hdd";
    if (/router|openwrt|gateway|wifi/.test(t)) return "router";
    if (/mac|macbook|imac|apple/.test(t)) return "monitor";
    return "server";
  }

  function hostAccent(h) {
    if (h.kind === "ikuai") return "t";
    var t = ((h.id || "") + " " + ((h.labels || []).join(" "))).toLowerCase();
    if (/nas|synology|dsm|storage/.test(t)) return "v";
    if (/openwrt|router|gateway|wifi/.test(t)) return "s";
    if (/proxmox|pve/.test(t)) return "p";
    return "c";
  }

  function hostAccentOf(id) {
    var h = hostMap[id];
    return h ? hostAccent(h) : "c";
  }

  function hostNameOf(id) {
    var h = hostMap[id];
    return h ? h.name : String(id || "");
  }

  function hostState(h) {
    if (!h.online) return "off";
    var mem = pct(h.mem_used, h.mem_total);
    var disk = h.disk_total ? pct(h.disk_used, h.disk_total) : 0;
    if (h.cpu >= 90 || mem >= 92 || disk >= 92) return "crit";
    if (h.cpu >= 75 || mem >= 85 || disk >= 85) return "warn";
    return "ok";
  }

  /* ================= markup builders ================= */

  function cell(inner, w, h) {
    return '<div class="cell" style="width:' + w + '%;height:' + h + '%">' + inner + "</div>";
  }

  function beam() { return '<div class="beam"><i></i><i></i></div>'; }

  function barRow(k, tone) {
    return '<div class="bl"><span class="bl-k">' + k + '</span><span class="bl-b"><i class="' +
      (tone || "") + '" style="width:0%"></i></span><span class="bl-v">--</span></div>';
  }

  function statRow(k, cls) {
    return '<div class="stat' + (cls ? " " + cls : "") + '"><div class="stat-k">' + k +
      '</div><div class="stat-v">--</div></div>';
  }

  function sparkSvg(acc) {
    var color = ACC[acc] || C.mauve;
    return '<div class="spark"><div class="spark-cap"><b>CPU</b><span>6 MIN</span></div>' +
      '<svg viewBox="0 0 100 34" preserveAspectRatio="none">' +
      '<polygon class="sp-a" fill="' + color + '" opacity="0.07" points=""></polygon>' +
      '<line class="sp-t" x1="0" y1="3.4" x2="100" y2="3.4" stroke="' + C.red + '" stroke-width="0.6" ' +
      'stroke-dasharray="2,2" opacity="0"></line>' +
      '<polyline class="sp-l" fill="none" stroke="' + color + '" stroke-width="1.3" ' +
      'stroke-linejoin="round" points=""></polyline>' +
      "</svg></div>";
  }

  // Auto-scaled so a 2% idle line is still readable: top is the smallest
  // round number >= peak*1.25, never below 25%.
  function sparkTop(list) {
    var max = 0, i;
    for (i = 0; i < list.length; i++) max = Math.max(max, clamp(list[i]));
    var top = Math.ceil(max * 1.25 / 5) * 5;
    if (top < 25) top = 25;
    if (top > 100) top = 100;
    return top;
  }

  // Anchored to the right edge so the line grows over the 6-minute window
  // instead of stretching two samples across the whole tile.
  function sparkGeom(list, top) {
    var N = 36, n = list.length, pts = [], i, x, y;
    if (!n) return { line: "", area: "" };
    if (n === 1) {
      y = (34 - clamp(list[0]) * 34 / top).toFixed(1);
      pts = ["99," + y, "100," + y];
    } else {
      for (i = 0; i < n; i++) {
        // stretch the samples we have across the tile; the ring never holds
        // more than N points, so this is the whole 6-minute window once full
        x = i * 100 / (n - 1);
        y = 34 - clamp(list[i]) * 34 / top;
        pts.push(x.toFixed(2) + "," + y.toFixed(1));
      }
    }
    var line = pts.join(" ");
    var x0 = pts[0].split(",")[0];
    var x1 = pts[pts.length - 1].split(",")[0];
    return { line: line, area: x0 + ",34 " + line + " " + x1 + ",34" };
  }

  /* ================= view (stable DOM) ================= */

  function view(id, items, key, build, update) {
    var el = $(id);
    if (!el) return;
    var k = items.length + "|" + key;
    if (el._k !== k) {
      var html = "", i;
      for (i = 0; i < items.length; i++) html += build(items[i], i);
      el.innerHTML = html || '<div class="empty">NO DATA</div>';
      el._k = k;
    }
    if (!items.length) return;
    var cells = el.children;
    for (var j = 0; j < items.length; j++) {
      var c = cells[j];
      if (!c || !c.firstElementChild) continue;
      update(c.firstElementChild, items[j], j);
    }
  }

  function gridSize(n, cols) {
    var rows = Math.max(1, Math.ceil(n / cols));
    return { w: 100 / cols, h: 100 / rows, rows: rows };
  }

  /* ================= overview ================= */

  function setStat(r, i, key, value, unit) {
    var v = r.statsV[i];
    if (!v) return;
    var k = r.statsK[i];
    if (k && k.textContent !== key) k.textContent = key;
    v.innerHTML = value + (unit ? "<small>" + unit + "</small>" : "");
  }

  function setBar(r, i, value, text, neutral) {
    var el = r.bars[i], v = r.barV[i];
    if (!el) return;
    var val = clamp(value);
    el.style.width = val + "%";
    el.className = neutral ? "b-neu" : (val >= 90 ? "b-crit" : (val >= 80 ? "b-warn" : ""));
    if (v) v.textContent = text;
  }

  function fmtTime(ts) {
    var d = ts ? new Date(ts * 1000) : new Date();
    var h = d.getHours(), m = d.getMinutes();
    return (h < 10 ? "0" : "") + h + ":" + (m < 10 ? "0" : "") + m;
  }

  // Overview answers one question - "is anything wrong?" - and nothing else.
  // Per-host numbers live on the HOSTS page.
  function renderOverview() {
    var hosts = snapshot.hosts || [];
    var ctrs = snapshot.containers || [];
    var pve = snapshot.pve || { nodes: [], guests: [] };
    var svcs = snapshot.services || [];
    var alerts = snapshot.alerts || [];
    var guests = pve.guests || [];
    var online = 0, offline = [], i;
    for (i = 0; i < hosts.length; i++) {
      if (hosts[i].online) online++; else offline.push(hosts[i].name);
    }
    var runCtr = 0, upGuest = 0, okSvc = 0;
    for (i = 0; i < ctrs.length; i++) if (ctrs[i].state === "running") runCtr++;
    for (i = 0; i < guests.length; i++) if (guests[i].status === "running") upGuest++;
    for (i = 0; i < svcs.length; i++) if (svcs[i].ok) okSvc++;

    var kp = [
      { k: "HOSTS", v: online + "/" + hosts.length,
        s: offline.length ? "OFFLINE · " + offline.join(" · ") : "ALL ONLINE", a: "c" },
      { k: "CONTAINERS", v: runCtr + "/" + ctrs.length, s: "RUNNING", a: "v" },
      { k: "GUESTS", v: upGuest + "/" + guests.length, s: "QEMU / LXC RUNNING", a: "p" },
      { k: "SERVICES", v: okSvc + "/" + svcs.length, s: "PROBE OK", a: "s" }
    ];
    var html = "", j, k;
    for (i = 0; i < kp.length; i++) {
      k = kp[i];
      html += '<div class="kpi a-' + k.a + '"><div class="kpi-k">' + k.k +
        '</div><div class="kpi-v">' + k.v + '</div><div class="kpi-sub">' + esc(k.s) + "</div></div>";
    }
    $("kpis").innerHTML = html;

    var ah;
    if (!alerts.length) {
      ah = '<div class="allclear"><div class="allclear-dot"></div>' +
        '<div class="allclear-t">ALL SYSTEMS NOMINAL</div>' +
        '<div class="allclear-s">' + hosts.length + " HOSTS · " + ctrs.length + " CONTAINERS · " +
        svcs.length + " SERVICES · CHECKED " + fmtTime(snapshot.ts) + "</div></div>";
    } else {
      ah = '<div class="alerts-h">ALERTS<span class="cnt">' + alerts.length + '</span></div>' +
        '<div class="alerts-list">';
      for (j = 0; j < alerts.length; j++) {
        var lv = alerts[j].level === "error" ? "crit" : "warn";
        ah += '<div class="alert-row"><span class="mk">' + marker(lv, 12) + "</span>" +
          esc(alerts[j].text) + '<span class="lv">' +
          (alerts[j].level === "error" ? "ERROR" : "WARN") + "</span></div>";
      }
      ah += "</div>";
    }
    $("alerts").innerHTML = ah;
  }

  // Cells of a partially filled last row stretch to use the whole width.
  function fillRow(el, n, cols, h) {
    var cells = el.children;
    var rows = Math.max(1, Math.ceil(n / cols));
    var lastStart = (rows - 1) * cols;
    var lastCount = n - lastStart;
    for (var i = 0; i < cells.length; i++) {
      var w = i >= lastStart ? 100 / Math.max(1, lastCount) : 100 / cols;
      cells[i].style.width = w + "%";
      cells[i].style.height = h + "%";
    }
  }

  function hostKey(hosts) {
    var out = [];
    for (var i = 0; i < hosts.length; i++) out.push(hosts[i].id);
    return out.join(",");
  }

  /* ================= hosts (instrument rows) ================= */

  function hostCols(n) { return n <= 1 ? 1 : 2; }

  function sparkSvgMini(acc) {
    var color = ACC[acc] || C.mauve;
    return '<svg viewBox="0 0 100 34" preserveAspectRatio="none">' +
      '<polygon class="sp-a" fill="' + color + '" opacity="0.08" points=""></polygon>' +
      '<polyline class="sp-l" fill="none" stroke="' + color + '" stroke-width="1.3" ' +
      'stroke-linejoin="round" points=""></polyline></svg>';
  }

  function buildHostCard(h) {
    var st = hostState(h);
    var acc = hostAccent(h);
    var n = (snapshot.hosts || []).length;
    var g = gridSize(n, hostCols(n));
    var router = h.kind === "ikuai";
    var html = '<div class="hcard a-' + acc + ' ' + st + '">';
    html += '<div class="hcard-top"><span class="hcard-ico">' +
      icon(hostIcon(h), 30, iconColor(st, acc)) + "</span>";
    html += '<div class="hcard-id"><div class="hcard-name">' + esc(h.name) + "</div>";
    html += '<div class="hcard-sub">--</div></div><span class="sp"></span>' +
      '<span class="mk">' + marker(st) + "</span></div>";
    html += '<div class="hcard-spark">' + sparkSvgMini(acc) + "</div>";
    html += '<div class="hcard-stats">' + statRow("CPU") + statRow("MEM") +
      statRow(router ? "CLIENTS" : "LOAD") + statRow(router ? "CONNS" : "DISK") + "</div>";
    html += '<div class="hcard-net"><span class="dn">↓ --</span><span class="up">↑ --</span></div>';
    html += "</div>";
    return cell(html, g.w, g.h);
  }

  function updateHostCard(tile, h) {
    var st = hostState(h);
    var acc = hostAccent(h);
    var r = refs(tile);
    if (tile._st !== st || tile._acc !== acc) {
      tile._st = st;
      tile._acc = acc;
      tile.className = "hcard a-" + acc + " " + st;
    }
    if (r.icon) r.icon.setAttribute("stroke", iconColor(st, acc));
    if (r.marker) r.marker.innerHTML = marker(st);
    if (r.name) r.name.textContent = h.name;
    if (r.sub) {
      var where = h.kind === "ikuai" ? ("WAN " + (h.wan_ip || "--")) : h.address;
      r.sub.textContent = where + (h.online ? " · UP " + fmtUptime(h.uptime) : " · OFFLINE");
    }

    var series = h.cpu_hist || [];
    var top = sparkTop(series);
    var geom = sparkGeom(series, top);
    if (r.spark) {
      r.spark.setAttribute("points", geom.line);
      if (r.sparkA) r.sparkA.setAttribute("points", series.length >= 6 ? geom.area : "");
    }

    var mem = h.online ? pct(h.mem_used, h.mem_total) : 0;
    if (r.statsV.length === 4) {
      setStat(r, 0, "CPU", h.online ? fmtPct(h.cpu) : "--", "");
      setStat(r, 1, "MEM", h.online ? fmtPct(mem) : "--", "");
      if (h.kind === "ikuai") {
        setStat(r, 2, "CLIENTS", h.online ? String(h.clients || 0) : "--", "");
        setStat(r, 3, "CONNS", h.online ? String(h.conns || 0) : "--", "");
      } else {
        setStat(r, 2, "LOAD", h.online && h.load1 ? h.load1.toFixed(2) : "--", "");
        setStat(r, 3, "DISK", h.online && h.disk_total ? fmtPct(pct(h.disk_used, h.disk_total)) : "--", "");
      }
    }
    var noNet = h.kind === "macos";
    if (r.netD) {
      r.netD.textContent = "↓ " + (h.online && !noNet
        ? (h.kind === "ikuai" ? fmtRate(h.wan_down) : fmtRate(h.net_rx)) : "--");
    }
    if (r.netU) {
      r.netU.textContent = "↑ " + (h.online && !noNet
        ? (h.kind === "ikuai" ? fmtRate(h.wan_up) : fmtRate(h.net_tx)) : "--");
    }
  }

  function renderHosts() {
    var hosts = snapshot.hosts || [];
    var g = gridSize(hosts.length, hostCols(hosts.length));
    view("grid-hosts", hosts, hostKey(hosts) + "|" + hostCols(hosts.length), buildHostCard, updateHostCard);
    var cells = $("grid-hosts").children;
    for (var i = 0; i < cells.length; i++) {
      cells[i].style.width = g.w + "%";
      cells[i].style.height = g.h + "%";
    }
  }

  /* ================= containers ================= */

  function buildContainer(c) {
    var running = c.state === "running";
    var st = running ? "ok" : "off";
    var acc = hostAccentOf(c.host);
    var catAcc = catAccent(c.name + " " + c.image);
    var g = gridSize((snapshot.containers || []).length, 6);
    var html = '<div class="tile sm ' + st + ' a-' + acc + '">';
    html += '<div class="tile-top">' +
      iconFor(c.name + " " + c.image, 20, running ? ACC[catAcc] : C.off, "box") +
      '<span class="sp"></span><span class="mk">' + marker(st, 10) + "</span></div>";
    html += '<div class="sm-name">' + esc(c.name) + "</div>";
    html += '<div class="sm-sub"><span class="c-host" style="color:' + ACC[acc] + '">' +
      esc(hostNameOf(c.host)) + '</span><span class="c-stat">--</span></div>';
    html += '<div class="sm-bar"><i style="width:0%"></i></div>';
    return cell(html + "</div>", g.w, g.h);
  }

  function updateContainer(tile, c) {
    var running = c.state === "running";
    var st = running ? "ok" : "off";
    var acc = hostAccentOf(c.host);
    var r = refs(tile);
    if (tile._st !== st || tile._acc !== acc) {
      tile._st = st;
      tile._acc = acc;
      tile.className = "tile sm " + st + " a-" + acc;
    }
    if (r.cstat) r.cstat.textContent = running ? " · " + fmtPct(c.cpu) + " · " + fmtBytes(c.mem) : " · STOPPED";
    if (r.smBar) {
      r.smBar.style.width = clamp(c.cpu) + "%";
      r.smBar.style.background = c.cpu >= 90 ? C.red : (c.cpu >= 70 ? C.yellow : "#9399b2");
    }
  }

  function containerKey(list) {
    var out = [];
    for (var i = 0; i < list.length; i++) out.push(list[i].name);
    return out.join(",");
  }

  function renderContainers() {
    var list = snapshot.containers || [];
    view("grid-containers", list, containerKey(list) + "|6", buildContainer, updateContainer);
  }

  /* ================= PVE ================= */

  function pveNodes() { return (snapshot.pve && snapshot.pve.nodes) || []; }
  function pveGuests() { return (snapshot.pve && snapshot.pve.guests) || []; }

  function nodeCols(n) { return n <= 2 ? 2 : (n <= 6 ? 3 : 4); }
  function guestCols(n) { return n <= 8 ? 4 : (n <= 12 ? 5 : 6); }

  function buildPVENode(n) {
    var st = n.online ? (n.cpu >= 90 ? "crit" : (n.cpu >= 75 ? "warn" : "ok")) : "off";
    var g = gridSize(pveNodes().length, nodeCols(pveNodes().length));
    var gs = pveGuests(), running = 0, total = 0, i;
    for (i = 0; i < gs.length; i++) {
      if (gs[i].node !== n.name) continue;
      total++;
      if (gs[i].status === "running") running++;
    }
    var html = '<div class="tile big tilt ' + st + ' a-p">' + beam();
    html += '<div class="t-head"><div class="tile-top"><span class="ico">' +
      icon("layers", 34, iconColor(st, "p")) + '</span><span class="sp"></span><span class="mk">' +
      marker(st) + "</span></div>";
    html += '<div class="t-name">' + esc(n.name) + "</div>";
    html += '<div class="t-sub">PVE NODE · ' + running + "/" + total + " GUESTS UP</div></div>";
    html += '<div class="t-mid"><div class="bars">' + barRow("CPU") + barRow("MEM") + "</div></div>";
    html += '<div class="t-bot"><div class="stats">' + statRow("UPTIME", "w50") +
      statRow("RAM", "w50") + "</div></div></div>";
    return cell(html, g.w, g.h);
  }

  function updatePVENode(tile, n) {
    var st = n.online ? (n.cpu >= 90 ? "crit" : (n.cpu >= 75 ? "warn" : "ok")) : "off";
    var r = refs(tile);
    if (tile._st !== st) { tile._st = st; tile.className = "tile big tilt " + st + " a-p"; }
    if (r.marker) r.marker.innerHTML = marker(st);
    if (r.icon) r.icon.setAttribute("stroke", iconColor(st, "p"));
    if (r.sub) {
      var gs = pveGuests(), running = 0, total = 0, i;
      for (i = 0; i < gs.length; i++) {
        if (gs[i].node !== n.name) continue;
        total++;
        if (gs[i].status === "running") running++;
      }
      r.sub.textContent = "PVE NODE · " + running + "/" + total + " GUESTS UP";
    }
    var mem = n.online ? pct(n.mem_used, n.mem_total) : 0;
    setBar(r, 0, n.online ? n.cpu : 0, n.online ? fmtPct(n.cpu) : "OFF");
    setBar(r, 1, mem, n.online ? fmtPct(mem) : "--", true);
    if (r.statsV.length === 2) {
      setStat(r, 0, "UPTIME", n.online ? fmtUptime(n.uptime) : "--", "");
      setStat(r, 1, "RAM", fmtBytes(n.mem_total), "");
    }
  }

  function buildPVEGuest(gg) {
    var up = gg.status === "running";
    var gst = up ? "ok" : "off";
    var g = gridSize(pveGuests().length, guestCols(pveGuests().length));
    var h = '<div class="tile big tilt ' + gst + ' a-v">' + beam();
    h += '<div class="t-head"><div class="tile-top"><span class="ico">' +
      icon(gg.kind === "lxc" ? "box" : "monitor", 30, iconColor(gst, "v")) +
      '</span><span class="sp"></span><span class="mk">' + marker(gst) + "</span></div>";
    h += '<div class="t-name">' + gg.id + " · " + esc(gg.name) + "</div>";
    h += '<div class="t-sub">' + (gg.kind === "lxc" ? "LXC" : "QEMU") + " · " +
      (up ? "RUNNING" : "STOPPED") + " · " + esc(gg.node) + "</div></div>";
    h += '<div class="t-mid"><div class="bars">' + barRow("CPU") +
      '<div class="bl"><span class="bl-k">MEM</span><span class="bl-b"><i class="b-neu" style="width:0%"></i></span>' +
      '<span class="bl-v">--</span></div></div></div>';
    h += '<div class="t-bot"><div class="foot"><span>RAM <b class="g-ram">--</b></span>' +
      '<span>DISK <b class="g-disk">--</b></span></div></div></div>';
    return cell(h, g.w, g.h);
  }

  function updatePVEGuest(tile, gg) {
    var up = gg.status === "running";
    var gst = up ? "ok" : "off";
    var r = refs(tile);
    if (tile._st !== gst) { tile._st = gst; tile.className = "tile big tilt " + gst + " a-v"; }
    if (r.marker) r.marker.innerHTML = marker(gst);
    if (r.icon) r.icon.setAttribute("stroke", iconColor(gst, "v"));
    if (r.sub) {
      r.sub.textContent = (gg.kind === "lxc" ? "LXC" : "QEMU") + " · " +
        (up ? "RUNNING" : "STOPPED") + " · " + gg.node;
    }
    setBar(r, 0, up ? gg.cpu : 0, up ? fmtPct(gg.cpu) : "OFF");
    if (r.bars[1]) r.bars[1].style.width = up ? clamp(pct(gg.mem, gg.maxmem)) + "%" : "0%";
    if (r.barV[1]) r.barV[1].textContent = up ? fmtBytes(gg.mem) : "--";
    var ram = tile.querySelector(".g-ram");
    var disk = tile.querySelector(".g-disk");
    if (ram) ram.textContent = up ? fmtBytes(gg.mem) : "--";
    if (disk) disk.textContent = gg.maxdisk ? fmtBytes(gg.maxdisk) : "--";
  }

  function nodeKey(list) {
    var out = [];
    for (var i = 0; i < list.length; i++) out.push(list[i].name);
    return out.join(",") + "|" + nodeCols(list.length);
  }

  function guestKey(list) {
    var out = [];
    for (var i = 0; i < list.length; i++) out.push(list[i].id);
    return out.join(",") + "|" + guestCols(list.length);
  }

  function renderPVE() {
    var nodes = pveNodes(), guests = pveGuests();
    view("grid-pve-nodes", nodes, nodeKey(nodes), buildPVENode, updatePVENode);
    view("grid-pve-guests", guests, guestKey(guests), buildPVEGuest, updatePVEGuest);
  }

  /* ================= services ================= */

  function serviceIcon(s) {
    var n = String(s.name || "").toLowerCase();
    if (/pve|proxmox/.test(n)) return "layers";
    if (/nas|synology|dsm/.test(n)) return "hdd";
    if (/openwrt|ikuai|router/.test(n)) return "router";
    for (var i = 0; i < RULES.length; i++) {
      if (RULES[i][0].test(n)) return RULES[i][1];
    }
    return "globe";
  }

  function serviceCols() {
    var n = (snapshot.services || []).length;
    return n <= 4 ? 2 : 3;
  }

  function buildService(s) {
    var st = s.ok ? "ok" : "crit";
    var acc = catAccent(serviceIcon(s));
    var g = gridSize((snapshot.services || []).length, serviceCols());
    var html = '<div class="tile big tilt ' + st + ' a-s">' + beam();
    html += '<div class="t-head"><div class="tile-top"><span class="ico">' +
      icon(serviceIcon(s), 38, st === "crit" ? C.red : (ACC[acc] || C.neutral)) +
      '</span><span class="sp"></span><span class="mk">' + marker(st) + "</span></div>";
    html += '<div class="t-name">' + esc(s.name) + "</div>";
    html += '<div class="t-sub">' + esc(s.url) + "</div></div>";
    html += '<div class="t-mid"><div class="svc-hero"><div class="svc-k">RESPONSE</div>' +
      '<div class="svc-v">--<small>MS</small></div></div></div>';
    html += '<div class="t-bot"><div class="svc-verdict"><span class="mk">' + marker(st) +
      '</span><span class="svc-word">--</span><span class="svc-meta"></span></div></div></div>';
    return cell(html, g.w, g.h);
  }

  function updateService(tile, s) {
    var st = s.ok ? "ok" : "crit";
    var r = refs(tile);
    if (tile._st !== st) { tile._st = st; tile.className = "tile big tilt " + st + " a-s"; }
    if (r.icon) {
      var acc = catAccent(serviceIcon(s));
      r.icon.setAttribute("stroke", st === "crit" ? C.red : (ACC[acc] || C.neutral));
    }
    if (r.svcV) {
      var ms = (s.latency_ms === undefined || s.latency_ms === null) ? "--" : s.latency_ms;
      r.svcV.innerHTML = ms + "<small>MS</small>";
    }
    var marks = tile.querySelectorAll(".mk");
    for (var i = 0; i < marks.length; i++) marks[i].innerHTML = marker(st);
    if (r.svcWord) r.svcWord.textContent = s.ok ? "ONLINE" : "DOWN";
    if (r.svcMeta) {
      r.svcMeta.innerHTML = "<span>" + (s.code || "---") + "</span><span>" +
        (s.cert_days !== null && s.cert_days !== undefined ? "TLS " + s.cert_days + "d" : "") + "</span>";
    }
  }

  function serviceKey(list) {
    var out = [];
    for (var i = 0; i < list.length; i++) out.push(list[i].name);
    return out.join(",");
  }

  function renderServices() {
    var list = snapshot.services || [];
    view("grid-services", list, serviceKey(list) + "|" + serviceCols(), buildService, updateService);
  }

  /* ================= trend ================= */

  function loadTrend() {
    var hosts = (snapshot && snapshot.hosts) || [];
    for (var i = 0; i < hosts.length; i++) {
      (function (id) {
        xhr("api/history?host=" + encodeURIComponent(id) + "&range=6h", function (err, data) {
          if (err) return;
          hist[id] = (data && data.samples) || [];
          renderTrend();
        });
      })(hosts[i].id);
    }
  }

  function renderTrend() {
    var body = $("trend-body");
    if (!body) return;
    var hosts = (snapshot && snapshot.hosts) || [];
    if (!hosts.length) { body.innerHTML = '<div class="empty">NO DATA</div>'; return; }
    var rows = Math.max(1, Math.ceil(hosts.length / 2));
    var per = 100 / rows;
    var html = "";
    var lastStart = (rows - 1) * 2;
    var lastCount = hosts.length - lastStart;
    for (var i = 0; i < hosts.length; i++) {
      var h = hosts[i];
      var samples = hist[h.id] || [];
      var mem = pct(h.mem_used, h.mem_total);
      var w = i >= lastStart ? 100 / Math.max(1, lastCount) : 50;
      html += '<div class="chart" style="width:' + w + '%;height:' + per + '%"><div class="chart-in">';
      html += '<div class="chart-h"><span class="chart-name">' + esc(h.name) + "</span>";
      html += '<span class="chart-v">' + fmtPct(h.cpu) + " · " + fmtPct(mem) + "</span></div>";
      html += chartSvg(samples);
      html += "</div></div>";
    }
    body.innerHTML = html;
  }

  function chartSvg(samples) {
    if (!samples || samples.length < 2) {
      return '<div class="chart-empty">COLLECTING · 1 POINT / MIN</div>';
    }
    var Wd = 320, H = 100, n = samples.length, i, s;
    var cpu = [], mem = [], area = [];
    for (i = 0; i < n; i++) {
      s = samples[i];
      var x = (i * Wd / (n - 1)).toFixed(1);
      cpu.push(x + "," + (H - clamp(s.cpu) * H / 100).toFixed(1));
      mem.push(x + "," + (H - clamp(pct(s.mem_used, s.mem_total)) * H / 100).toFixed(1));
    }
    area = ["0," + H].concat(cpu).concat([Wd + "," + H]);

    var g = "";
    for (i = 0; i <= 4; i++) {
      var y = (i * H / 4).toFixed(1);
      g += '<line x1="0" y1="' + y + '" x2="' + Wd + '" y2="' + y + '" stroke="#26263a" stroke-width="1"/>';
    }
    g += '<line x1="0" y1="' + (H * 0.1).toFixed(1) + '" x2="' + Wd + '" y2="' + (H * 0.1).toFixed(1) +
      '" stroke="' + C.red + '" stroke-width="1" stroke-dasharray="4,4" opacity="0.55"/>';

    return '<div class="chart-plot"><svg viewBox="0 0 ' + Wd + " " + H + '" preserveAspectRatio="none">' + g +
      '<polygon fill="' + C.mauve + '" opacity="0.08" points="' + area.join(" ") + '"/>' +
      '<polyline fill="none" stroke="' + C.teal + '" stroke-width="1.4" points="' + mem.join(" ") + '"/>' +
      '<polyline fill="none" stroke="' + C.mauve + '" stroke-width="1.6" points="' + cpu.join(" ") + '"/>' +
      "</svg></div>";
  }

  /* ================= refresh ================= */

  var bootPoll = 0;

  function refresh() {
    xhr("api/snapshot", function (err, data) {
      if (err) {
        $("offline-banner").className = "show";
        return;
      }
      $("offline-banner").className = "";
      snapshot = data;
      if (data.hosts && data.hosts.length) hideBoot();
      else if (!bootOff) bootText("WAITING FOR FIRST SAMPLE");
      try {
        render();
      } catch (e) {
        if (window.console && console.log) console.log("render error:", e);
      }
      // Right after a server restart the first snapshot is empty; poll fast
      // until the first collection lands instead of waiting a full cycle.
      if ((!data.hosts || !data.hosts.length) && bootPoll < 12) {
        bootPoll++;
        setTimeout(refresh, 1200);
      }
      var now = new Date().getTime();
      if (now - lastTrend > TREND_MS) {
        lastTrend = now;
        loadTrend();
      }
    });
  }

  function indexHosts() {
    hostMap = {};
    var hs = snapshot.hosts || [];
    for (var i = 0; i < hs.length; i++) hostMap[hs[i].id] = hs[i];
  }

  function render() {
    if (!snapshot) return;
    indexHosts();
    renderSys();
    renderOverview();
    renderHosts();
    renderContainers();
    renderPVE();
    renderServices();
    if (pageIndex === 5) renderTrend();
  }

  function renderSys() {
    var hosts = snapshot.hosts || [];
    var ctrs = snapshot.containers || [];
    var alerts = snapshot.alerts || [];
    var online = 0, running = 0, i;
    for (i = 0; i < hosts.length; i++) if (hosts[i].online) online++;
    for (i = 0; i < ctrs.length; i++) if (ctrs[i].state === "running") running++;
    $("sys").innerHTML = "HOST <b>" + online + "/" + hosts.length + "</b>" +
      "&nbsp;·&nbsp;CTR <b>" + running + "/" + ctrs.length + "</b>" +
      "&nbsp;·&nbsp;ALERT <b" + (alerts.length ? ' class="bad"' : "") + ">" + alerts.length + "</b>";
  }

  /* ================= ambient effects ================= */

  var spotEl = null;
  var spotTimer = null;

  function spotlight(x, y) {
    if (!spotEl) spotEl = $("spotlight");
    if (!spotEl) return;
    spotEl.style.webkitTransform = "translate3d(" + x + "px," + y + "px,0)";
    spotEl.className = "on";
    if (spotTimer) clearTimeout(spotTimer);
    spotTimer = null;
  }

  function spotlightOff() {
    if (spotTimer) clearTimeout(spotTimer);
    spotTimer = setTimeout(function () {
      if (spotEl) spotEl.className = "";
    }, 260);
  }

  var tiltEl = null;

  function tiltBegin(el) {
    tiltEl = el;
    if (el) el.className = el.className + " tilting";
  }

  function tiltMove(el, x, y) {
    if (!el || !el.getBoundingClientRect) return;
    var r = el.getBoundingClientRect();
    var dx = (x - (r.left + r.width / 2)) / (r.width / 2);
    var dy = (y - (r.top + r.height / 2)) / (r.height / 2);
    if (dx > 1) dx = 1; if (dx < -1) dx = -1;
    if (dy > 1) dy = 1; if (dy < -1) dy = -1;
    el.style.webkitTransform = "perspective(720px) rotateX(" + (dy * -3.2).toFixed(2) +
      "deg) rotateY(" + (dx * 3.2).toFixed(2) + "deg) translateZ(0)";
  }

  function tiltEnd() {
    if (!tiltEl) return;
    var el = tiltEl;
    tiltEl = null;
    el.className = el.className.replace(/\s*tilting/g, "");
    el.style.webkitTransform = "";
  }

  /* ================= navigation ================= */

  function pageEl(i) { return $("page-" + PAGES[i].id); }

  function setLabel(i) {
    var el = $("section");
    if (!el) return;
    el.style.color = ACC[PAGES[i].acc] || C.dim;
    el.textContent = PAGES[i].label;
  }

  function updateDots() {
    var html = "";
    for (var i = 0; i < PAGES.length; i++) {
      html += '<div class="dot-item' + (i === pageIndex ? " on" : "") + '"></div>';
    }
    var dots = $("dots");
    dots.innerHTML = html;
    var on = dots.firstChild;
    if (on) {
      var col = ACC[PAGES[pageIndex].acc] || C.mauve;
      on.style.background = col;
      on.style.webkitBoxShadow = "0 0 8px " + col + ", 0 0 18px " + col;
    }
  }

  var slideTimer = null;

  function showPage(target, dir) {
    target = ((target % PAGES.length) + PAGES.length) % PAGES.length;
    if (target === pageIndex) return;
    dir = dir || (target > pageIndex ? 1 : -1);

    var cur = pageEl(pageIndex);
    var nxt = pageEl(target);
    var p;

    for (p = 0; p < PAGES.length; p++) {
      var el = pageEl(p);
      if (el && el !== cur && el !== nxt) el.className = "page";
    }
    if (cur) {
      cur.style.webkitTransform = "";
      cur.style.opacity = "";
    }
    nxt.style.webkitTransform = "";
    nxt.style.opacity = "";
    nxt.className = "page";
    void nxt.offsetWidth;
    nxt.className = dir > 0 ? "page enter-r" : "page enter-l";
    void nxt.offsetWidth;
    cur.className = dir > 0 ? "page leave-l" : "page leave-r";
    nxt.className = "page active";

    pageIndex = target;
    setLabel(target);
    updateDots();

    if (slideTimer) clearTimeout(slideTimer);
    slideTimer = setTimeout(function () {
      if (cur) cur.className = "page";
      nxt.className = "page active";
    }, 400);
    pauseUntil = new Date().getTime() + 8000;
  }

  function showPageNow(i) {
    pageIndex = i;
    for (var p = 0; p < PAGES.length; p++) {
      var el = pageEl(p);
      if (!el) continue;
      el.className = p === i ? "page active" : "page";
      el.style.webkitTransform = "";
      el.style.opacity = "";
    }
    var s = $("section");
    if (s) {
      s.textContent = PAGES[i].label;
      s.style.color = ACC[PAGES[i].acc] || C.dim;
    }
    updateDots();
  }

  /* ---- finger-following swipe ---- */

  var drag = null;

  function dragInit(x, y) {
    drag = {
      x: x, y: y, startX: x, startY: y,
      t0: new Date().getTime(), lastT: new Date().getTime(),
      dir: 0, mode: "", cur: pageEl(pageIndex), nxt: null
    };
    W = $("pages").offsetWidth || window.innerWidth;
  }

  function dragStart(dir) {
    var cur = pageEl(pageIndex);
    var ni = pageIndex + dir;
    if (ni < 0 || ni >= PAGES.length) { drag.nxt = null; drag.dir = dir; return; }
    var nxt = pageEl(ni);
    for (var p = 0; p < PAGES.length; p++) {
      var el = pageEl(p);
      if (el && el !== cur && el !== nxt) el.className = "page";
    }
    nxt.style.display = "block";
    nxt.style.webkitTransition = "none";
    nxt.style.opacity = "0.6";
    nxt.style.webkitTransform = "translate3d(" + (dir * W) + "px,0,0)";
    nxt.className = "page dragging";
    nxt.style.display = "block";
    nxt.style.webkitTransition = "none";
    drag.nxt = nxt;
    drag.dir = dir;
    cur.className = "page active dragging";
    cur.style.webkitTransition = "none";
    cur.style.display = "block";
    drag.cur = cur;
  }

  function dragMove(dx) {
    if (!drag) return;
    var cur = drag.cur, nxt = drag.nxt;
    var d = dx;
    if (!nxt && ((pageIndex === 0 && dx > 0) || (pageIndex === PAGES.length - 1 && dx < 0))) {
      d = dx * 0.32;
    }
    cur.style.webkitTransform = "translate3d(" + d + "px,0,0)";
    if (nxt) {
      nxt.style.webkitTransform = "translate3d(" + (d + drag.dir * W) + "px,0,0)";
      nxt.style.opacity = String(0.6 + 0.4 * Math.min(1, Math.abs(d) / W));
    }
  }

  var EASE = "-webkit-transform .3s cubic-bezier(.22,.61,.36,1), opacity .3s linear";

  function dragEnd(commit) {
    if (!drag) return;
    var cur = drag.cur || pageEl(pageIndex);
    var nxt = drag.nxt;
    var dir = drag.dir;
    var ni = pageIndex + dir;

    cur.style.webkitTransition = EASE;
    if (nxt) nxt.style.webkitTransition = EASE;
    if (commit && nxt) {
      cur.style.webkitTransform = "translate3d(" + (-dir * W) + "px,0,0)";
      cur.style.opacity = "0";
      nxt.style.webkitTransform = "translate3d(0,0,0)";
      nxt.style.opacity = "1";
      pageIndex = ni;
      setLabel(ni);
      updateDots();
    } else {
      cur.style.webkitTransform = "translate3d(0,0,0)";
      cur.style.opacity = "1";
      if (nxt) {
        nxt.style.webkitTransform = "translate3d(" + (dir * W) + "px,0,0)";
        nxt.style.opacity = "0";
      }
    }

    drag = null;
    setTimeout(function () {
      var c = pageEl(pageIndex);
      for (var p = 0; p < PAGES.length; p++) {
        var el = pageEl(p);
        if (!el) continue;
        el.style.webkitTransition = "";
        el.style.opacity = "";
        el.style.webkitTransform = "";
        el.style.display = "";
        el.className = p === pageIndex ? "page active" : "page";
      }
      pauseUntil = new Date().getTime() + 60000;
    }, 330);
  }

  function bindTouch() {
    var lastX = 0, lastY = 0;

    document.body.addEventListener("touchstart", function (e) {
      if (pickerOpen) return;
      if (!e.touches || !e.touches.length) return;
      if (e.touches.length > 1) { drag = null; tiltEnd(); return; }
      var t = e.touches[0];
      lastX = t.clientX;
      lastY = t.clientY;
      dragInit(t.clientX, t.clientY);
      spotlight(t.clientX, t.clientY);
      var el = e.target;
      while (el && el !== document.body) {
        var cn = String(el.className || "");
        if (cn.indexOf("tile") >= 0 && cn.indexOf("big") >= 0) {
          tiltBegin(el);
          break;
        }
        el = el.parentNode;
      }
    }, false);

    document.body.addEventListener("touchmove", function (e) {
      if (!e.touches || !e.touches.length || !drag) return;
      var t = e.touches[0];
      var dx = t.clientX - drag.startX;
      var dy = t.clientY - drag.startY;
      drag.x = t.clientX;
      drag.y = t.clientY;
      lastX = t.clientX;
      lastY = t.clientY;
      spotlight(t.clientX, t.clientY);

      if (!drag.mode) {
        if (Math.abs(dx) > 10 && Math.abs(dx) > Math.abs(dy) * 1.1) {
          drag.mode = "swipe";
          if (tiltEl) tiltEnd();
          dragStart(dx < 0 ? 1 : -1);
        } else if (Math.abs(dy) > 14) {
          drag.mode = "scroll";
        }
      }

      // Always swallow the move: the panel never scrolls, and iOS otherwise
      // claims the gesture (rubber-band) and stops feeding us touchmove.
      if (e.preventDefault) e.preventDefault();

      if (drag.mode === "swipe") dragMove(dx);
      else if (tiltEl) tiltMove(tiltEl, t.clientX, t.clientY);
    }, false);

    document.body.addEventListener("touchend", function (e) {
      spotlightOff();
      tiltEnd();
      if (!drag) return;
      var t = (e.changedTouches && e.changedTouches[0]) || null;
      var dx = t ? t.clientX - drag.startX : lastX - drag.startX;
      var dy = t ? t.clientY - drag.startY : lastY - drag.startY;
      var dt = new Date().getTime() - drag.t0;
      var v = Math.abs(dx) / Math.max(1, dt);
      if (drag.mode === "swipe") {
        dragEnd(Math.abs(dx) > W * 0.16 || v > 0.35);
      } else if (Math.abs(dx) > 50 && Math.abs(dx) > Math.abs(dy) * 1.2) {
        // Fallback for Safari versions that swallow touchmove during gesture
        // negotiation: decide on the end point instead of the live drag.
        var dir = dx < 0 ? 1 : -1;
        drag = null;
        showPage(pageIndex + dir, dir);
        pauseUntil = new Date().getTime() + 60000;
      } else {
        drag = null;
      }
    }, false);

    document.body.addEventListener("touchcancel", function () {
      tiltEnd();
      spotlightOff();
      if (!drag) return;
      if (drag.mode === "swipe") dragEnd(false); else drag = null;
    }, false);
  }

  /* ================= fps probe / degradation ================= */

  function fpsProbe() {
    var last = 0, slow = 0, done = false;
    function tick(t) {
      if (done) return;
      if (last) {
        var dt = t - last;
        if (dt > 42) slow++; else slow = Math.max(0, slow - 1);
        if (slow >= 45) {
          document.body.className = (document.body.className + " lowfx").replace(/^\s+/, "");
          done = true;
          return;
        }
      }
      last = t;
      requestAnimationFrame(tick);
    }
    requestAnimationFrame(tick);
  }

  /* ================= clock ================= */

  function clock() {
    var d = new Date();
    var h = d.getHours(), m = d.getMinutes();
    $("clock").innerHTML = (h < 10 ? "0" : "") + h + ":" + (m < 10 ? "0" : "") + m;
  }

  /* ================= init ================= */

  /* ================= background style ================= */

  function findBg(id) {
    for (var i = 0; i < BGS.length; i++) if (BGS[i].id === id) return BGS[i];
    return BGS[0];
  }

  function savedBgId() {
    try { return localStorage.getItem("paddeck.bg"); } catch (e) { return null; }
  }

  function applyBg(id) {
    var b = findBg(id);
    var el = $("bgfx");
    if (el) el.className = b.cls;
    var grid = $("bg-grid");
    if (grid) grid.style.display = b.id === "retro" ? "block" : "none";
    try { localStorage.setItem("paddeck.bg", b.id); } catch (e) {}
  }

  /* ================= boot screen ================= */

  var bootOff = false;

  function hideBoot() {
    if (bootOff) return;
    bootOff = true;
    var el = $("boot");
    if (el) el.className = "hide";
  }

  function bootText(t) {
    var el = $("boot-s");
    if (el && el.textContent !== t) el.textContent = t;
  }

  /* ================= theme picker ================= */

  var pickerOpen = true;
  var pickerTimer = null;

  function savedThemeId() {
    try { return localStorage.getItem("paddeck.theme"); } catch (e) { return null; }
  }
  function saveThemeId(id) {
    try { localStorage.setItem("paddeck.theme", id); } catch (e) {}
  }
  function findTheme(id) {
    for (var i = 0; i < THEMES.length; i++) if (THEMES[i].id === id) return THEMES[i];
    return THEMES[0];
  }
  function applyTheme(id) {
    var t = findTheme(id);
    if (!t) return;
    setTheme(t);
    var link = $("theme");
    if (link) link.href = "theme-" + t.id + ".css";
    saveThemeId(t.id);
  }
  function closePicker() {
    pickerOpen = false;
    var el = $("picker");
    if (el) el.className = "hide";
    if (pickerTimer) { clearInterval(pickerTimer); pickerTimer = null; }
    pauseUntil = new Date().getTime() + 3000;
  }
  function buildPickerBgSelection(id) {
    var bgs = $("pk-bgs");
    if (!bgs) return;
    var items = bgs.querySelectorAll(".pk-bg-in");
    for (var i = 0; i < items.length; i++) {
      var sel = items[i].getAttribute("data-bg") === id;
      items[i].className = "pk-bg-in " + findBg(items[i].getAttribute("data-bg")).cls + (sel ? " sel" : "");
      items[i].style.borderColor = sel && currentTheme ? currentTheme.a1 : "";
    }
  }

  function buildPicker() {
    var grid = $("pk-grid");
    if (!grid) { pickerOpen = false; return; }
    var saved = savedThemeId() || "mocha";
    var html = "", i, j;
    for (i = 0; i < THEMES.length; i++) {
      var t = THEMES[i];
      var chips = "";
      for (j = 1; j <= 6; j++) chips += '<i style="background:' + t["a" + j] + '"></i>';
      html += '<div class="pk-item"><div class="pk-in' + (t.id === saved ? " sel" : "") +
        '" data-id="' + t.id + '" style="background:' + t.panel + ";border-color:" +
        (t.id === saved ? t.a1 : t.border) + '">' +
        '<div class="pk-name" style="color:' + t.text + '">' + t.name + "</div>" +
        '<div class="pk-desc" style="color:' + t.sub + '">' + t.desc + "</div>" +
        '<div class="pk-prev" style="background:' + t.bg + ";border-color:" + t.border + '">' +
        '<div class="pk-prev-t" style="background:' + t.panel + ";border-left-color:" + t.a1 + '">' +
        '<i style="background:' + t.text + '"></i>' +
        '<i style="background:' + t.sub + '"></i></div>' +
        '<div class="pk-prev-b"><b style="background:' + t.a2 + '"></b>' +
        '<b style="background:' + t.a3 + '"></b><b style="background:' + t.a5 + '"></b></div></div>' +
        '<div class="pk-chips">' + chips + "</div></div></div>";
    }
    grid.innerHTML = html;
    grid.onclick = function (ev) {
      var el = ev.target;
      while (el && el !== grid) {
        var id = el.getAttribute && el.getAttribute("data-id");
        if (id) { applyTheme(id); closePicker(); return; }
        el = el.parentNode;
      }
    };
    var bgs = $("pk-bgs");
    if (bgs) {
      var bhtml = "", bi, b;
      var savedBg = savedBgId() || "aurora";
      for (bi = 0; bi < BGS.length; bi++) {
        b = BGS[bi];
        bhtml += '<div class="pk-bg"><div class="pk-bg-in ' + b.cls +
          (b.id === savedBg ? " sel" : "") + '" data-bg="' + b.id + '" style="background-color:' +
          (currentTheme ? currentTheme.bg : "#1e1e2e") + '">' +
          '<div class="pk-bg-t">' + b.name + "</div></div></div>";
      }
      bgs.innerHTML = bhtml;
      bgs.onclick = function (ev) {
        var el = ev.target;
        while (el && el !== bgs) {
          var id = el.getAttribute && el.getAttribute("data-bg");
          if (id) { applyBg(id); buildPickerBgSelection(id); return; }
          el = el.parentNode;
        }
      };
    }

    var left = savedThemeId() ? 15 : 40;
    var foot = $("pk-foot");
    function label() {
      foot.textContent = "TAP · " + left + "s 后继续（" +
        (savedThemeId() ? "上次配色" : "默认配色") + "）";
    }
    label();
    pickerTimer = setInterval(function () {
      left--;
      if (left <= 0) { closePicker(); return; }
      label();
    }, 1000);
  }

  function init() {
    // Standalone (home-screen) mode draws under the 20px iOS status bar.
    if (window.navigator && window.navigator.standalone) {
      document.body.className = (document.body.className + " standalone").replace(/^\s+/, "");
    }
    // ?theme=gruvbox&bg=aurora 直接跳过选择页（书签 / 自动化用）
    var qs = location.search || "";
    var qTheme = /[?&]theme=([a-z]+)/.exec(qs);
    var qBg = /[?&]bg=([a-z]+)/.exec(qs);
    applyTheme(qTheme ? qTheme[1] : (savedThemeId() || "mocha"));
    applyBg(qBg ? qBg[1] : (savedBgId() || "aurora"));
    buildPicker();
    if (qTheme || qBg) closePicker();

    var start = 0;
    var hash = (location.hash || "").replace("#", "");
    for (var i = 0; i < PAGES.length; i++) if (PAGES[i].id === hash) start = i;

    // ?lowfx forces the degraded profile (for comparing on a slow iPad).
    if (location.search.indexOf("lowfx") >= 0) {
      document.body.className = (document.body.className + " lowfx").replace(/^\s+/, "");
    }

    showPageNow(start);
    bindTouch();
    clock();
    setInterval(clock, 10000);
    refresh();
    setInterval(refresh, REFRESH_MS);
    setInterval(function () {
      if (new Date().getTime() < pauseUntil) return;
      showPage(pageIndex + 1, 1);
    }, PAGE_MS);
    setTimeout(function () { loadTrend(); }, 3000);
    setTimeout(fpsProbe, 6000);
  }

  if (document.readyState === "complete" || document.readyState === "interactive") {
    init();
  } else {
    document.addEventListener("DOMContentLoaded", init, false);
  }
})();
