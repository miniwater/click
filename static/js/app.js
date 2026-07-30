(() => {
  const $ = (id) => document.getElementById(id);
  const goldEl = $("gold");
  const cpsEl = $("cps");
  const diaEl = $("diamonds");
  const onlineEl = $("online");
  const meEl = $("me");
  const clickBtn = $("clickBtn");
  const stage = $("stage");
  const clickPowerEl = $("clickPower");
  const clickLv = $("clickLv");
  const clickCost = $("clickCost");
  const upgradeClickBtn = $("upgradeClickBtn");
  const facilityList = $("facilityList");
  const chatLog = $("chatLog");
  const chatForm = $("chatForm");
  const chatInput = $("chatInput");
  const toasts = $("toasts");

  let state = null;
  let myName = "";
  let myColor = "#6c8cff";
  let ws = null;
  let reconnectTimer = null;
  let clickBurst = 0;
  let clickBurstTimer = null;
  let spaceFlushTimer = null;
  let buyHoldTimer = null;
  let buyRepeatTimer = null;
  let suppressBuyClick = false;
  let upgradeHoldTimer = null;
  let upgradeRepeatTimer = null;
  let suppressUpgradeClick = false;

  function fmt(n) {
    const value = normalizeDecimal(n);
    const [integer, fraction = ""] = value.split(".");
    if (integer.length <= 3) {
      const decimals = integer.length === 3 ? 0 : integer.length === 2 ? 1 : 2;
      const tail = fraction.slice(0, decimals).padEnd(decimals, "0");
      return decimals ? `${integer}.${tail}` : integer;
    }

    const group = Math.floor((integer.length - 1) / 3);
    const leading = integer.length - group * 3;
    const digits = (integer + fraction).slice(0, leading + 2).padEnd(leading + 2, "0");
    const number = digits.slice(0, leading) + "." + digits.slice(leading);
    return number + unitName(group);
  }

  function normalizeDecimal(value) {
    let text = String(value == null ? "0" : value).trim();
    if (!/^\d+(\.\d+)?$/.test(text)) return "0";
    let [integer, fraction = ""] = text.split(".");
    integer = integer.replace(/^0+(?=\d)/, "");
    fraction = fraction.replace(/0+$/, "");
    return fraction ? `${integer}.${fraction}` : integer;
  }

  function compareDecimal(a, b) {
    const [ai, af = ""] = normalizeDecimal(a).split(".");
    const [bi, bf = ""] = normalizeDecimal(b).split(".");
    if (ai.length !== bi.length) return ai.length > bi.length ? 1 : -1;
    if (ai !== bi) return ai > bi ? 1 : -1;
    const width = Math.max(af.length, bf.length);
    const ap = af.padEnd(width, "0");
    const bp = bf.padEnd(width, "0");
    return ap === bp ? 0 : ap > bp ? 1 : -1;
  }

  function unitName(group) {
    const common = ["", "K", "M", "B", "T", "Qa", "Qi", "Sx", "Sp", "Oc", "No"];
    if (group < common.length) return common[group];

    // Excel-style alphabetic units give an unbounded, deterministic suffix sequence.
    let index = group - common.length;
    let suffix = "";
    do {
      suffix = String.fromCharCode(65 + (index % 26)) + suffix;
      index = Math.floor(index / 26) - 1;
    } while (index >= 0);
    return "U" + suffix;
  }

  function applyState(s, fromClick) {
    if (!s) return;
    state = s;
    goldEl.textContent = fmt(s.gold);
    diaEl.textContent = fmt(s.diamonds);
    cpsEl.textContent = fmt(s.cps) + "/s";
    clickPowerEl.textContent = "+" + fmt(s.clickPower);
    clickLv.textContent = "Lv." + s.clickLevel;
    clickCost.textContent = "🪙 " + fmt(s.clickCost);
    const canUpgradeClick = compareDecimal(s.gold, s.clickCost) >= 0;
    upgradeClickBtn.disabled = !canUpgradeClick;
    upgradeClickBtn.classList.toggle("afford", canUpgradeClick);
    renderFacilities();
  }

  function renderFacilities() {
    if (!state) return;
    const defs = state.facilityDefs || [];
    const facs = state.facilities || [];
    const map = {};
    facs.forEach((f) => (map[f.id] = f));
    facilityList.innerHTML = defs
      .map((d) => {
        const f = map[d.id] || { owned: 0, enhance: 0, cost: d.baseCost, cps: 0, unitCps: d.baseCps };
        const canBuy = compareDecimal(state.gold, f.cost) >= 0;
        const canEnh = state.diamonds >= 1 && f.owned > 0;
        return `<div class="card" data-id="${d.id}">
          <div class="row">
            <span class="ico">${d.icon}</span>
            <div class="info">
              <div class="name">${d.name} <span style="color:var(--muted);font-weight:500">×${f.owned}</span></div>
              <div class="desc">${d.desc}</div>
              <div class="facility-detail">
                <span>单位 ${fmt(f.unitCps)}/s</span>
                <span>强化 +${f.enhance}</span>
              </div>
            </div>
            <button type="button" class="buy-btn ${canBuy ? "afford" : ""}" data-act="buy" data-id="${d.id}" ${canBuy ? "" : "disabled"}>购买</button>
            <button type="button" class="enh-btn" data-act="enhance" data-id="${d.id}" title="消耗1钻石，收益×1.01" ${canEnh ? "" : "disabled"}>💎</button>
          </div>
           <div class="meta-line">
             <span>产出 ${fmt(f.cps)}/s</span>
             <span>🪙 ${fmt(f.cost)}</span>
           </div>
            ${f.owned > 0 ? `
            <div class="status">
              <span class="status-dot"></span>
              <span>正在运行</span>
            </div>` : ''}
        </div>`;
      })
      .join("");
  }

  facilityList.addEventListener("click", (e) => {
    const btn = e.target.closest("button[data-act]");
    if (!btn || btn.disabled) return;
    const id = +btn.dataset.id;
    if (btn.dataset.act === "buy") {
      if (suppressBuyClick) {
        suppressBuyClick = false;
        return;
      }
      send({ type: "buy", id });
    }
    if (btn.dataset.act === "enhance") send({ type: "enhance", id });
  });

  function stopBuyHold() {
    clearTimeout(buyHoldTimer);
    clearInterval(buyRepeatTimer);
    buyHoldTimer = null;
    buyRepeatTimer = null;
  }

  facilityList.addEventListener("pointerdown", (e) => {
    const btn = e.target.closest('button[data-act="buy"]');
    if (!btn || btn.disabled) return;

    const id = +btn.dataset.id;
    suppressBuyClick = false;
    stopBuyHold();
    buyHoldTimer = setTimeout(() => {
      suppressBuyClick = true;
      send({ type: "buy", id });
      buyRepeatTimer = setInterval(() => send({ type: "buy", id }), 150);
    }, 400);
  });

  window.addEventListener("pointerup", stopBuyHold);
  window.addEventListener("pointercancel", stopBuyHold);
  window.addEventListener("blur", stopBuyHold);

  upgradeClickBtn.addEventListener("click", () => {
    if (suppressUpgradeClick) {
      suppressUpgradeClick = false;
      return;
    }
    send({ type: "upgrade_click" });
  });

  function stopUpgradeHold() {
    clearTimeout(upgradeHoldTimer);
    clearInterval(upgradeRepeatTimer);
    upgradeHoldTimer = null;
    upgradeRepeatTimer = null;
  }

  upgradeClickBtn.addEventListener("pointerdown", () => {
    if (upgradeClickBtn.disabled) return;
    suppressUpgradeClick = false;
    stopUpgradeHold();
    upgradeHoldTimer = setTimeout(() => {
      suppressUpgradeClick = true;
      send({ type: "upgrade_click" });
      upgradeRepeatTimer = setInterval(() => {
        if (upgradeClickBtn.disabled) {
          stopUpgradeHold();
          return;
        }
        send({ type: "upgrade_click" });
      }, 150);
    }, 400);
  });

  window.addEventListener("pointerup", stopUpgradeHold);
  window.addEventListener("pointercancel", stopUpgradeHold);
  window.addEventListener("blur", stopUpgradeHold);

  function floatText(text, x, y, cls) {
    const el = document.createElement("div");
    el.className = "float-num" + (cls ? " " + cls : "");
    el.textContent = text;
    el.style.left = x + "px";
    el.style.top = y + "px";
    stage.appendChild(el);
    setTimeout(() => el.remove(), 900);
  }

  function doLocalClick(ev, deferFlush) {
    if (!ws || ws.readyState !== 1) return;
    clickBurst++;
    clickBtn.classList.remove("bounce");
    void clickBtn.offsetWidth;
    clickBtn.classList.add("bounce");

    const rect = stage.getBoundingClientRect();
    const cx = (ev.clientX || rect.left + rect.width / 2) - rect.left;
    const cy = (ev.clientY || rect.top + rect.height / 2) - rect.top;
    const power = state ? state.clickPower : 1;
    floatText("+" + fmt(power), cx - 10 + (Math.random() * 30 - 15), cy - 10, "");

    if (!deferFlush) {
      clearTimeout(clickBurstTimer);
      clickBurstTimer = setTimeout(flushClicks, 80);
    }
  }

  function flushClicks() {
    if (clickBurst <= 0) return;
    const n = clickBurst;
    clickBurst = 0;
    for (let remaining = n; remaining > 0; remaining -= 20) {
      send({ type: "click", n: Math.min(remaining, 20) });
    }
  }

  clickBtn.addEventListener("pointerdown", (e) => {
    e.preventDefault();
    doLocalClick(e);
  });

  function stopSpaceHold() {
    clearInterval(spaceFlushTimer);
    spaceFlushTimer = null;
    flushClicks();
  }

  // 键盘空格连点按固定周期结算，避免自动重复按键一直推迟发送。
  window.addEventListener("keydown", (e) => {
    if (e.code === "Space" && document.activeElement !== chatInput) {
      e.preventDefault();
      if (!spaceFlushTimer) {
        clearTimeout(clickBurstTimer);
        clickBurstTimer = null;
        spaceFlushTimer = setInterval(flushClicks, 1000);
      }
      doLocalClick({ clientX: 0, clientY: 0 }, true);
    }
  });
  window.addEventListener("keyup", (e) => {
    if (e.code === "Space" && spaceFlushTimer) stopSpaceHold();
  });
  window.addEventListener("blur", stopSpaceHold);

  chatForm.addEventListener("submit", (e) => {
    e.preventDefault();
    const text = chatInput.value.trim();
    if (!text) return;
    send({ type: "chat", text });
    chatInput.value = "";
  });

  function appendChat(html, scroll) {
    const div = document.createElement("div");
    div.innerHTML = html;
    while (div.firstChild) chatLog.appendChild(div.firstChild);
    if (scroll !== false) chatLog.scrollTop = chatLog.scrollHeight;
  }

  function addChatRow(c) {
    appendChat(
      `<div class="chat-item"><span class="who" style="color:${esc(c.color)}">${esc(c.name)}</span>${esc(c.text)}</div>`
    );
  }

  function addNotify(text, name, color) {
    appendChat(
      `<div class="chat-item notify" style="border-left:3px solid ${esc(color || "#6c8cff")};padding-left:8px">${esc(text)}</div>`
    );
    toast(text);
  }

  function toast(text, err) {
    while (toasts.children.length >= 4) {
      toasts.firstElementChild.remove();
    }
    const t = document.createElement("div");
    t.className = "toast" + (err ? " err" : "");
    t.textContent = text;
    toasts.appendChild(t);
    setTimeout(() => {
      if (t.isConnected) t.remove();
    }, 2800);
  }

  function esc(s) {
    return String(s == null ? "" : s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function send(obj) {
    if (ws && ws.readyState === 1) ws.send(JSON.stringify(obj));
  }

  function connect() {
    const proto = location.protocol === "https:" ? "wss:" : "ws:";
    ws = new WebSocket(proto + "//" + location.host + "/ws");
    ws.onopen = () => {
      meEl.textContent = "已连接";
    };
    ws.onclose = () => {
      meEl.textContent = "重连中…";
      clearTimeout(reconnectTimer);
      reconnectTimer = setTimeout(connect, 1500);
    };
    ws.onerror = () => ws.close();
    ws.onmessage = (ev) => {
      let msg;
      try {
        msg = JSON.parse(ev.data);
      } catch {
        return;
      }
      switch (msg.type) {
        case "welcome":
          myName = msg.name;
          myColor = msg.color;
          document.documentElement.style.setProperty("--me", myColor);
          meEl.textContent = myName;
          meEl.style.color = myColor;
          applyState(msg.state);
          onlineEl.textContent = "在线 " + (msg.online || 0);
          chatLog.innerHTML = "";
          (msg.chats || []).forEach(addChatRow);
          break;
        case "state":
          applyState(msg.state);
          break;
        case "click_result":
          applyState(msg.state, true);
          if (msg.diamondsGot > 0) {
            const rect = stage.getBoundingClientRect();
            floatText("💎+" + msg.diamondsGot, rect.width / 2, rect.height / 2 - 40, "dia");
            if (msg.name === myName) toast("幸运！获得钻石 ×" + msg.diamondsGot);
          }
          break;
        case "chat":
          addChatRow(msg.chat);
          break;
        case "notify":
          addNotify(msg.text, msg.name, msg.color);
          break;
        case "online":
          onlineEl.textContent = "在线 " + (msg.online || 0);
          break;
        case "error":
          toast(msg.text, true);
          break;
      }
    };
  }

  connect();
})();
