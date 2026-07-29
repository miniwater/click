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
  let displayGold = 0;
  let lastServerGold = 0;
  let lastServerCPS = 0;
  let lastSync = performance.now();
  let clickBurst = 0;
  let clickBurstTimer = null;
  let buyHoldTimer = null;
  let buyRepeatTimer = null;
  let suppressBuyClick = false;
  let upgradeHoldTimer = null;
  let upgradeRepeatTimer = null;
  let suppressUpgradeClick = false;

  function fmt(n) {
    if (n == null || isNaN(n)) return "0";
    const abs = Math.abs(n);
    if (abs >= 1e12) return (n / 1e12).toFixed(2) + "T";
    if (abs >= 1e9) return (n / 1e9).toFixed(2) + "B";
    if (abs >= 1e6) return (n / 1e6).toFixed(2) + "M";
    if (abs >= 1e4) return (n / 1e3).toFixed(2) + "K";
    if (abs >= 100) return n.toFixed(0);
    if (abs >= 10) return n.toFixed(1);
    return n.toFixed(2);
  }

  function applyState(s, fromClick) {
    if (!s) return;
    state = s;
    lastServerGold = s.gold;
    lastServerCPS = s.cps || 0;
    lastSync = performance.now();
    if (!fromClick) displayGold = s.gold;
    diaEl.textContent = fmt(s.diamonds);
    cpsEl.textContent = fmt(s.cps) + "/s";
    clickPowerEl.textContent = "+" + fmt(s.clickPower);
    clickLv.textContent = "Lv." + s.clickLevel;
    clickCost.textContent = "🪙 " + fmt(s.clickCost);
    upgradeClickBtn.disabled = s.gold < s.clickCost;
    upgradeClickBtn.classList.toggle("afford", s.gold >= s.clickCost);
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
        const canBuy = state.gold >= f.cost;
        const canEnh = state.diamonds >= 1 && f.owned > 0;
        return `<div class="card" data-id="${d.id}">
          <div class="row">
            <span class="ico">${d.icon}</span>
            <div class="info">
              <div class="name">${d.name} <span style="color:var(--muted);font-weight:500">×${f.owned}</span></div>
              <div class="desc">${d.desc} · 单位 ${fmt(f.unitCps)}/s · 强化+${f.enhance}</div>
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

  function doLocalClick(ev) {
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

    clearTimeout(clickBurstTimer);
    clickBurstTimer = setTimeout(flushClicks, 80);
  }

  function flushClicks() {
    if (clickBurst <= 0) return;
    const n = clickBurst;
    clickBurst = 0;
    send({ type: "click", n });
  }

  clickBtn.addEventListener("pointerdown", (e) => {
    e.preventDefault();
    doLocalClick(e);
  });

  // 键盘空格连点
  window.addEventListener("keydown", (e) => {
    if (e.code === "Space" && document.activeElement !== chatInput) {
      e.preventDefault();
      doLocalClick({ clientX: 0, clientY: 0 });
    }
  });

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
    const t = document.createElement("div");
    t.className = "toast" + (err ? " err" : "");
    t.textContent = text;
    toasts.appendChild(t);
    setTimeout(() => t.remove(), 2800);
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
          displayGold = msg.state.gold;
          onlineEl.textContent = "在线 " + (msg.online || 0);
          chatLog.innerHTML = "";
          (msg.chats || []).forEach(addChatRow);
          break;
        case "state":
          applyState(msg.state);
          break;
        case "click_result":
          applyState(msg.state, true);
          displayGold = msg.state.gold;
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

  // 本地平滑显示自动产出的金币。
  function frame(now) {
    if (state) {
      const dt = (now - lastSync) / 1000;
      const predicted = lastServerGold + lastServerCPS * dt;
      displayGold += (predicted - displayGold) * 0.25;
      goldEl.textContent = fmt(displayGold);

    }
    requestAnimationFrame(frame);
  }

  connect();
  requestAnimationFrame(frame);
})();
