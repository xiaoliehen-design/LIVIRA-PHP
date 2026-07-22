(() => {
  "use strict";

  const $ = (selector, root = document) => root ? root.querySelector(selector) : null;
  const $$ = (selector, root = document) => root ? [...root.querySelectorAll(selector)] : [];

  function initializeIdleSession() {
    if (!document.body.classList.contains("app-body")) return;
    const csrf = $('meta[name="csrf-token"]')?.content || "";
    const timeoutSeconds = Number($('meta[name="idle-timeout-seconds"]')?.content || 1800);
    if (!csrf || !Number.isFinite(timeoutSeconds) || timeoutSeconds <= 0) return;

    const timeoutMs = timeoutSeconds * 1000;
    const pingIntervalMs = Math.min(5 * 60 * 1000, Math.max(60 * 1000, timeoutMs / 4));
    const activityStorageKey = "livira:last-activity";
    let lastActivity = Date.now();
    let lastStorageWrite = 0;
    let lastPing = Date.now();
    let pingInFlight = false;
    let loggingOut = false;

    const sharedLastActivity = () => {
      try {
        const stored = Number(window.localStorage.getItem(activityStorageKey));
        return Number.isFinite(stored) && stored > 0 ? Math.max(stored, lastActivity) : lastActivity;
      } catch (_) {
        return lastActivity;
      }
    };

    const markActivity = () => {
      const now = Date.now();
      lastActivity = now;
      if (now - lastStorageWrite < 5000) return;
      lastStorageWrite = now;
      try {
        window.localStorage.setItem(activityStorageKey, String(now));
      } catch (_) {
        // Session timeout still works in this tab when storage is unavailable.
      }
    };

    const redirectToIdleLogin = () => {
      window.location.assign("/login?idle=1");
    };

    const endIdleSession = async () => {
      if (loggingOut) return;
      loggingOut = true;
      try {
        await fetch("/session/idle-logout", {
          method: "POST",
          headers: { "X-CSRF-Token": csrf, Accept: "application/json" },
          credentials: "same-origin",
          keepalive: true,
        });
      } catch (_) {
        // The next login-page request also invalidates an expired cookie.
      } finally {
        redirectToIdleLogin();
      }
    };

    const pingSession = async () => {
      if (pingInFlight || loggingOut) return;
      pingInFlight = true;
      try {
        const response = await fetch("/session/activity", {
          method: "POST",
          headers: { "X-CSRF-Token": csrf, Accept: "application/json" },
          credentials: "same-origin",
        });
        if (response.status === 401) {
          loggingOut = true;
          redirectToIdleLogin();
          return;
        }
        if (response.ok) lastPing = Date.now();
      } catch (_) {
        // A temporary network error must not destroy a still-valid session.
      } finally {
        pingInFlight = false;
      }
    };

    ["pointerdown", "keydown", "touchstart", "scroll", "mousemove"].forEach((eventName) => {
      window.addEventListener(eventName, markActivity, { passive: true });
    });
    document.addEventListener("visibilitychange", () => {
      if (!document.hidden) markActivity();
    });
    window.addEventListener("storage", (event) => {
      if (event.key === activityStorageKey) {
        const value = Number(event.newValue);
        if (Number.isFinite(value)) lastActivity = Math.max(lastActivity, value);
      }
    });

    markActivity();
    window.setInterval(() => {
      const now = Date.now();
      const activityAt = sharedLastActivity();
      if (now - activityAt >= timeoutMs) {
        endIdleSession();
        return;
      }
      if (activityAt > lastPing && now - lastPing >= pingIntervalMs) pingSession();
    }, 15000);
  }

  initializeIdleSession();

  function initializeLoginCaptcha() {
	const box = $("[data-captcha-box]");
	if (!box) return;
	const image = $("[data-captcha-image]", box);
	const token = $("[data-captcha-token]", box);
	const answer = $("[data-captcha-answer]", box);
	const refresh = $("[data-refresh-captcha]", box);
	if (!image || !token || !answer || !refresh) return;

	refresh.addEventListener("click", async () => {
	  refresh.disabled = true;
	  try {
		const response = await fetch("/captcha/new", { headers: { Accept: "application/json" }, credentials: "same-origin", cache: "no-store" });
		if (!response.ok) throw new Error("CAPTCHA unavailable");
		const challenge = await response.json();
		if (!challenge.token || !challenge.image_url) throw new Error("Invalid CAPTCHA response");
		token.value = challenge.token;
		image.src = challenge.image_url;
		answer.value = "";
		answer.focus();
	  } catch (_) {
		window.alert("Kode CAPTCHA baru belum dapat dibuat. Muat ulang halaman dan coba kembali.");
	  } finally {
		refresh.disabled = false;
	  }
	});
  }

  initializeLoginCaptcha();

  function setOpenState(open) {
    document.body.classList.toggle("modal-open", open);
  }

  function openModal(modal) {
    if (!modal) return;
    modal.hidden = false;
    setOpenState(true);
    const focusable = $("input:not([type=hidden]):not(:disabled), button:not(:disabled), select:not(:disabled), textarea:not(:disabled)", modal);
    window.setTimeout(() => focusable?.focus(), 30);
  }

  function closeModal(modal) {
    if (!modal) return;
    modal.hidden = true;
    if (!$(".modal:not([hidden]), .drawer:not([hidden])")) setOpenState(false);
  }

  function openDrawer(drawer) {
    if (!drawer) return;
    drawer.hidden = false;
    setOpenState(true);
  }

  function closeDrawer(drawer) {
    if (!drawer) return;
    drawer.hidden = true;
    if (!$(".modal:not([hidden]), .drawer:not([hidden])")) setOpenState(false);
  }

  $$('[data-dismiss]').forEach((button) => button.addEventListener("click", () => button.closest(".alert")?.remove()));
  $$('[data-delete-inventory-form]').forEach((form) => form.addEventListener("submit", (event) => {
    const reference = form.dataset.deleteReference || "data barang ini";
    const confirmed = window.confirm(`Hapus permanen ${reference}? Data tidak akan tampil lagi pada inventory, proses, pencarian, atau laporan. Snapshot audit tetap disimpan di database.`);
    if (!confirmed) event.preventDefault();
  }));
  $$('[data-delete-user-form]').forEach((form) => form.addEventListener("submit", (event) => {
	const name = form.dataset.userName || "pengguna ini";
	const email = form.dataset.userEmail || "email tidak tersedia";
	const confirmed = window.confirm(`Hapus permanen user ${name} (${email})? Akun tidak dapat login lagi dan harus mendaftar ulang jika ingin menggunakan LIVIRA.`);
	if (!confirmed) event.preventDefault();
  }));
  $$('[data-delete-role-form]').forEach((form) => form.addEventListener("submit", (event) => {
	const name = form.dataset.roleName || "role ini";
	const confirmed = window.confirm(`Hapus permanen role "${name}"? Role hanya dapat dihapus jika tidak sedang digunakan oleh pengguna dan tidak dapat dipulihkan setelah dihapus.`);
	if (!confirmed) event.preventDefault();
  }));

  const sidebar = $("[data-sidebar]");
  const overlay = $("[data-sidebar-overlay]");
  $("[data-sidebar-toggle]")?.addEventListener("click", () => {
    sidebar?.classList.toggle("open");
    overlay?.classList.toggle("open");
  });
  overlay?.addEventListener("click", () => {
    sidebar?.classList.remove("open");
    overlay.classList.remove("open");
  });

  const popoverToggles = $$('[data-popover-toggle]');

  function closeTopbarPopovers(exceptName = "") {
    $$('[data-popover]').forEach((popover) => {
      if (exceptName && popover.dataset.popover === exceptName) return;
      popover.hidden = true;
    });
    popoverToggles.forEach((toggle) => {
      if (exceptName && toggle.dataset.popoverToggle === exceptName) return;
      toggle.setAttribute("aria-expanded", "false");
    });
  }

  popoverToggles.forEach((toggle) => {
    toggle.addEventListener("click", (event) => {
      event.stopPropagation();
      const name = toggle.dataset.popoverToggle;
      const popover = $(`[data-popover="${name}"]`);
      if (!popover) return;
      const willOpen = popover.hidden;
      closeTopbarPopovers(willOpen ? name : "");
      popover.hidden = !willOpen;
      toggle.setAttribute("aria-expanded", String(willOpen));
    });
  });

  $$('[data-popover]').forEach((popover) => popover.addEventListener("click", (event) => event.stopPropagation()));
  document.addEventListener("click", () => closeTopbarPopovers());

  const profileModal = $("#profile-modal");
  $$('[data-open-profile-modal]').forEach((button) => {
    button.addEventListener("click", () => {
      closeTopbarPopovers();
      openModal(profileModal);
    });
  });

  $$('[data-open-process-dashboard]').forEach((button) => {
    button.addEventListener("click", () => openModal($(`#process-dashboard-${button.dataset.openProcessDashboard}`)));
  });

  const performanceDashboardModal = $("#performance-dashboard-modal");
  $$('[data-open-performance-dashboard]').forEach((button) => {
    button.addEventListener("click", () => openModal(performanceDashboardModal));
  });
  if (performanceDashboardModal?.dataset.autoOpen === "true") {
    openModal(performanceDashboardModal);
  }

  const capacityModal = $("#capacity-edit-modal");
  const capacityForm = $("[data-capacity-form]", capacityModal);
  const capacityFacility = $("[data-capacity-facility]", capacityModal);
  const yardCapacityInput = $("[data-yard-capacity-input]", capacityModal);
  const shedCapacityInput = $("[data-shed-capacity-input]", capacityModal);

  function syncCapacityEditor() {
    const option = capacityFacility?.selectedOptions?.[0];
    const facilityID = option?.value || "";
    if (yardCapacityInput) yardCapacityInput.value = option?.dataset.yardCapacity || "";
    if (shedCapacityInput) shedCapacityInput.value = option?.dataset.shedCapacity || "";
    if (capacityForm) capacityForm.action = facilityID ? `/admin/facilities/${encodeURIComponent(facilityID)}/capacity` : "";
  }

  capacityFacility?.addEventListener("change", syncCapacityEditor);
  $$('[data-open-capacity-editor]').forEach((button) => {
    button.addEventListener("click", () => {
      if (!capacityFacility) return;
      const requestedID = button.dataset.facilityId || "";
      const exists = requestedID && [...capacityFacility.options].some((option) => option.value === requestedID);
      capacityFacility.value = exists ? requestedID : capacityFacility.options[1]?.value || "";
      syncCapacityEditor();
      openModal(capacityModal);
    });
  });
  capacityForm?.addEventListener("submit", (event) => {
    if (!capacityFacility?.value || !capacityForm.action) {
      event.preventDefault();
      capacityFacility?.focus();
    }
  });

  $$('[data-auto-submit]').forEach((select) => select.addEventListener("change", () => select.form?.requestSubmit()));

  $$('[data-table-scroll-top]').forEach((topScroller) => {
    const panel = topScroller.closest('.table-panel, .report-results');
    const bodyScroller = $('[data-table-scroll-body]', panel);
    const table = $('table', bodyScroller);
    const spacer = $('div', topScroller);
    if (!bodyScroller || !table || !spacer) return;
    let syncing = false;
    const resize = () => {
      spacer.style.width = `${Math.max(table.scrollWidth, bodyScroller.clientWidth)}px`;
      topScroller.hidden = table.scrollWidth <= bodyScroller.clientWidth + 1;
    };
    const syncScroll = (source, target) => {
      if (syncing) return;
      syncing = true;
      target.scrollLeft = source.scrollLeft;
      window.requestAnimationFrame(() => { syncing = false; });
    };
    topScroller.addEventListener('scroll', () => syncScroll(topScroller, bodyScroller), { passive: true });
    bodyScroller.addEventListener('scroll', () => syncScroll(bodyScroller, topScroller), { passive: true });
    window.addEventListener('resize', resize, { passive: true });
    window.requestAnimationFrame(resize);
  });
  $$('[data-close-modal]').forEach((button) => button.addEventListener("click", () => closeModal(button.closest(".modal"))));
  $$('[data-close-drawer]').forEach((button) => button.addEventListener("click", () => closeDrawer(button.closest(".drawer"))));

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      $$(".modal:not([hidden])").forEach(closeModal);
      $$(".drawer:not([hidden])").forEach(closeDrawer);
      closeTopbarPopovers();
    }
    if (event.key === "/" && !["INPUT", "TEXTAREA", "SELECT"].includes(document.activeElement?.tagName)) {
      const search = $('input[type="search"]');
      if (search) {
        event.preventDefault();
        search.focus();
      }
    }
  });

  function formatDateTime(value) {
    if (!value) return "—";
    const date = new Date(value);
    if (Number.isNaN(date.getTime()) || date.getUTCFullYear() <= 1) return "—";
    return new Intl.DateTimeFormat("id-ID", { dateStyle: "medium", timeStyle: "short", timeZone: "Asia/Jakarta" }).format(date) + " WIB";
  }

  function formatDate(value) {
    if (!value) return "—";
    const date = new Date(value);
    if (Number.isNaN(date.getTime()) || date.getUTCFullYear() <= 1) return "—";
    return new Intl.DateTimeFormat("id-ID", { dateStyle: "medium", timeZone: "Asia/Jakarta" }).format(date);
  }

  function formatMoney(value) {
    const amount = Number(value || 0);
    return amount > 0 ? new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(amount) : "—";
  }

  const parameterForm = $("[data-parameter-form]");
  if (parameterForm) {
    const group = $("[data-parameter-group]", parameterForm);
    const scope = $("[data-parameter-scope]", parameterForm);
    const help = $("[data-parameter-help]", parameterForm);
    const label = $("[data-parameter-label]", parameterForm);
    const code = $("[data-parameter-code]", parameterForm);
    const hints = {
      bdn_category: ["Kategori khusus untuk penetapan BDN.", "Contoh: Barang lartas tertentu"],
      item_kind: ["Dipakai pada penetapan, pencacahan, pencarian, dan pelaporan.", "Contoh: Barang mudah rusak"],
      goods_condition: ["Kondisi fisik barang yang dicatat saat pencacahan dan dapat difilter pada laporan.", "Contoh: Baru"],
      unit: ["Satuan kuantitas barang pada penetapan dan pencacahan.", "Contoh: Container"],
      allocation_purpose: ["Pilihan usulan dan persetujuan peruntukan BMMN.", "Contoh: Penjualan secara lelang"],
      origin_tps: ["Master TPS asal pada pencatatan BTD/penetapan BDN.", "Contoh: PT Terminal Contoh"],
      tpp: ["Master TPP tujuan dan filter. Kode teknis menjadi ID TPP.", "Contoh: TPP Contoh Jakarta"],
      load_type: ["Jenis muatan pada manifest/kontainer.", "Contoh: FCL"],
      exit_type: ["Jenis pengeluaran; pilih cakupan BTD, BDN, BMMN, dan/atau Barang Titipan.", "Contoh: REEKSPOR"],
      transfer_type: ["Jenis serah terima pada proses Hibah/PSP.", "Contoh: Hibah"],
    };
    const updateParameterForm = () => {
      const value = group?.value || "";
      const hint = hints[value] || ["Pilih kelompok untuk melihat kegunaan parameter.", "Contoh nilai parameter"];
      if (help) help.textContent = hint[0];
      if (label) label.placeholder = hint[1];
      if (scope) scope.hidden = value !== "exit_type";
      if (scope?.hidden) {
        $$('.parameter-scope input[type="checkbox"]', parameterForm).forEach((checkbox) => { checkbox.checked = false; });
      }
      if (code) code.placeholder = value === "tpp" ? "Contoh: tpp-contoh-jakarta" : "Opsional; otomatis dari label";
    };
    group?.addEventListener("change", updateParameterForm);
    updateParameterForm();
  }

  const createInventoryModal = $("#inventory-create-modal");
  const inventoryManualForm = $("[data-inventory-manual-form]", createInventoryModal);
  const inventoryImportForm = $("[data-inventory-import-form]", createInventoryModal);
  const inventoryEntryModeButtons = $$('[data-inventory-entry-mode]', createInventoryModal);
  const inventoryExcelFile = $("[data-inventory-excel-file]", createInventoryModal);
  const inventoryFileName = $("[data-inventory-file-name]", createInventoryModal);
  const inventoryImportTitle = $("[data-import-title]", createInventoryModal);
  const inventoryTemplateLink = $("[data-import-template-link]", createInventoryModal);
  const inventoryTemplateConfig = {
    BTD: { href: "/templates/template_upload_btd.xlsx?v=1.0.8", label: "Unduh template BTD", title: "Upload banyak Pencatatan BTD" },
    BDN: { href: "/templates/template_upload_bdn.xlsx?v=1.0.8", label: "Unduh template BDN", title: "Upload banyak Penetapan BDN" },
    TITIPAN: { href: "/templates/template_upload_barang_titipan.xlsx?v=1.0.8", label: "Unduh template Barang Titipan", title: "Upload banyak Barang Titipan" },
  };

  function setInventoryEntryMode(mode) {
    const excelMode = mode === "excel";
    if (inventoryManualForm) inventoryManualForm.hidden = excelMode;
    if (inventoryImportForm) inventoryImportForm.hidden = !excelMode;
    inventoryEntryModeButtons.forEach((button) => {
      const active = button.dataset.inventoryEntryMode === mode;
      button.classList.toggle("active", active);
      button.setAttribute("aria-selected", active ? "true" : "false");
    });
  }

  inventoryEntryModeButtons.forEach((button) => button.addEventListener("click", () => {
    setInventoryEntryMode(button.dataset.inventoryEntryMode || "manual");
  }));

  inventoryExcelFile?.addEventListener("change", () => {
    const file = inventoryExcelFile.files?.[0];
    if (inventoryFileName) inventoryFileName.textContent = file ? `${file.name} · ${new Intl.NumberFormat("id-ID", { maximumFractionDigits: 1 }).format(file.size / 1024)} KB` : "Belum ada file dipilih · format .xlsx · maksimal 1.000 baris / 6 MB";
  });

  $$('[data-open-inventory-modal]').forEach((button) => {
    button.addEventListener("click", () => {
      const kind = button.dataset.openInventoryModal;
      inventoryManualForm?.reset();
      inventoryImportForm?.reset();
      setInventoryEntryMode("manual");
      if (inventoryFileName) inventoryFileName.textContent = "Belum ada file dipilih · format .xlsx · maksimal 1.000 baris / 6 MB";
      const isTitipan = kind === "TITIPAN";
      const isBTD = kind === "BTD";
      const title = $("#inventory-modal-title");
      const typeInputs = $$('[data-item-type-input]', createInventoryModal);
      const description = $("[data-inventory-modal-description]", createInventoryModal);
      const documentTitle = $("[data-document-fieldset-title]", createInventoryModal);
      const manifestTitle = $("[data-manifest-fieldset-title]", createInventoryModal);
      const noLabel = $("[data-determination-no-label]", createInventoryModal);
      const dateLabel = $("[data-determination-date-label]", createInventoryModal);
      const submit = $("[data-inventory-create-submit]", createInventoryModal);
      if (title) title.textContent = isTitipan ? "Pemasukan barang titipan kantor/unit lain" : isBTD ? "Pencatatan BTD" : `Penetapan ${kind}`;
      if (description) description.textContent = isTitipan ? "Catat barang titipan sesuai dokumen dasar pemasukan dan kondisi lokasi sebenarnya." : "Barang baru secara default dicatat masih berada di TPS.";
      if (documentTitle) documentTitle.textContent = isTitipan ? "Dokumen dasar pemasukan" : isBTD ? "Dokumen pencatatan BTD" : "Dokumen penetapan";
      if (manifestTitle) manifestTitle.textContent = isTitipan ? "BL opsional, manifest, dan muatan" : isBTD ? "BL, tanggal BL, manifest, muatan, dan TPS asal" : "BL opsional, manifest, muatan, dan TPS asal";
      if (noLabel) noLabel.textContent = isTitipan ? "Nomor dokumen dasar pemasukan" : isBTD ? "Nomor BTD" : "Nomor penetapan";
      if (dateLabel) dateLabel.textContent = isTitipan ? "Tanggal dokumen" : isBTD ? "Tanggal BTD" : "Tanggal penetapan";
      if (submit) submit.textContent = isTitipan ? "Simpan pemasukan" : isBTD ? "Simpan pencatatan" : "Simpan penetapan";
      typeInputs.forEach((input) => { input.value = kind; });
      const template = inventoryTemplateConfig[kind] || inventoryTemplateConfig.BTD;
      if (inventoryImportTitle) inventoryImportTitle.textContent = template.title;
      if (inventoryTemplateLink) {
        inventoryTemplateLink.href = template.href;
        const textNode = [...inventoryTemplateLink.childNodes].find((node) => node.nodeType === Node.TEXT_NODE);
        if (textNode) textNode.textContent = template.label;
        else inventoryTemplateLink.append(document.createTextNode(template.label));
      }
      const category = $("[data-bdn-category]", createInventoryModal);
      const categorySelect = $('select[name="category"]', category);
      const isBDN = kind === "BDN";
      if (category) category.hidden = !isBDN;
      if (categorySelect) {
        categorySelect.disabled = !isBDN;
        categorySelect.required = isBDN;
        if (!isBDN) categorySelect.value = "";
      }
      const entrustedFields = $("[data-entrusted-fields]", createInventoryModal);
      if (entrustedFields) entrustedFields.hidden = !isTitipan;
      $$('input, select, textarea', entrustedFields).forEach((field) => {
        field.disabled = !isTitipan;
        field.required = isTitipan;
        if (!isTitipan) field.value = "";
      });
      const originTPS = $("[data-origin-tps-field]", createInventoryModal);
      const originTPSSelect = $('select[name="origin_warehouse"]', originTPS);
      if (originTPS) originTPS.hidden = isTitipan;
      if (originTPSSelect) {
        originTPSSelect.disabled = isTitipan;
        originTPSSelect.required = !isTitipan;
        if (isTitipan) originTPSSelect.value = "";
      }
      const blField = $("[data-bl-field]", createInventoryModal);
      const blInput = $("[data-bl-input]", createInventoryModal);
      const blDateField = $("[data-bl-date-field]", createInventoryModal);
      const blDateInput = $("[data-bl-date-input]", createInventoryModal);
      const blRequiredMark = $("[data-bl-required-mark]", createInventoryModal);
      const blOptionalMark = $("[data-bl-optional-mark]", createInventoryModal);
      const blDateRequiredMark = $("[data-bl-date-required-mark]", createInventoryModal);
      const blDateOptionalMark = $("[data-bl-date-optional-mark]", createInventoryModal);
      if (blField) blField.hidden = false;
      if (blDateField) blDateField.hidden = false;
      if (blInput) {
        blInput.disabled = false;
        blInput.required = isBTD;
      }
      if (blDateInput) {
        blDateInput.disabled = false;
        blDateInput.required = isBTD;
      }
      if (blRequiredMark) blRequiredMark.hidden = !isBTD;
      if (blDateRequiredMark) blDateRequiredMark.hidden = !isBTD;
      if (blOptionalMark) blOptionalMark.hidden = isBTD;
      if (blDateOptionalMark) blDateOptionalMark.hidden = isBTD;
      const locationChoice = $(`input[name="at_tpp"][value="${isTitipan ? "sudah" : "belum"}"]`, createInventoryModal);
      if (locationChoice) locationChoice.checked = true;
      resetContainerEntries();
      updateLoadTypeFields();
      updateInitialLocationChoice();
      openModal(createInventoryModal);
    });
  });

  const loadTypeSelect = $("[data-load-type-select]", createInventoryModal);
  const fclSection = $("[data-fcl-container-section]", createInventoryModal);
  const lclSection = $("[data-lcl-volume-section]", createInventoryModal);
  const containerSizeDraft = $("[data-container-size-draft]", createInventoryModal);
  const containerNumberDraft = $("[data-container-number-draft]", createInventoryModal);
  const addContainerButton = $("[data-add-container]", createInventoryModal);
  const containersJSON = $("[data-containers-json]", createInventoryModal);
  const containerCardList = $("[data-container-card-list]", createInventoryModal);
  const containerCount = $("[data-container-count]", createInventoryModal);
  const lclVolumeInput = $("[data-lcl-volume-input]", createInventoryModal);
  const lclGoodsJSON = $("[data-lcl-goods-json]", createInventoryModal);
  const lclGoodsLines = $("[data-lcl-goods-lines]", createInventoryModal);
  const lclGoodsCount = $("[data-lcl-goods-count]", createInventoryModal);
  const addLCLGoodsButton = $("[data-add-lcl-goods-line]", createInventoryModal);
  const initialGoodsTemplate = $("[data-initial-goods-line-template]", createInventoryModal);
  let containerEntries = [];
  let lclGoods = [];

  function emptyGoodsLine() {
    return { description: "", item_kind: "", goods_value: "", quantity: "", quantity_detail: "", unit: "" };
  }

  function normalizeContainerDraft(value) {
    const compact = String(value || "").toUpperCase().replace(/[\s.\-]/g, "");
    if (!/^[A-Z]{4}\d{7}$/.test(compact)) return "";
    return `${compact.slice(0, 4)} ${compact.slice(4, 10)}-${compact.slice(10)}`;
  }

  function goodsLineValid(goods) {
    return !!goods.description.trim() && !!goods.item_kind && Number(goods.quantity) > 0 && !!goods.unit && (!goods.goods_value || Number(String(goods.goods_value).replace(/\D/g, "")) >= 0);
  }

  function buildGoodsLine(goods, index, onUpdate, onRemove, removable) {
    const fragment = initialGoodsTemplate?.content.cloneNode(true);
    const row = fragment?.querySelector("[data-initial-goods-line]");
    if (!row) return null;
    const title = $("[data-goods-line-title]", row);
    if (title) title.textContent = `Identitas barang ${index + 1}`;
    $$('[data-goods-field]', row).forEach((field) => {
      const key = field.dataset.goodsField;
      field.value = goods[key] ?? "";
      field.required = ["description", "item_kind", "quantity", "unit"].includes(key);
      field.addEventListener("input", () => {
        goods[key] = field.value;
        onUpdate();
      });
      field.addEventListener("change", () => {
        goods[key] = field.value;
        onUpdate();
      });
    });
    const remove = $("[data-remove-goods-line]", row);
    if (remove) {
      remove.hidden = !removable;
      remove.disabled = !removable;
      remove.addEventListener("click", onRemove);
    }
    return row;
  }

  function syncInitialGoodsJSON() {
    const serializeGoods = (goods) => ({ ...goods, quantity: Number(goods.quantity || 0) });
    if (containersJSON) containersJSON.value = JSON.stringify(containerEntries.map((entry) => ({ ...entry, goods: entry.goods.map(serializeGoods) })));
    if (lclGoodsJSON) lclGoodsJSON.value = JSON.stringify(lclGoods.map(serializeGoods));
    if (containerCount) containerCount.textContent = `${containerEntries.length} kontainer`;
    if (lclGoodsCount) lclGoodsCount.textContent = `${lclGoods.length} barang`;
  }

  function renderContainerEntries() {
    syncInitialGoodsJSON();
    if (!containerCardList) return;
    containerCardList.replaceChildren();
    if (!containerEntries.length) {
      const empty = document.createElement("p");
      empty.className = "container-list-empty";
      empty.textContent = "Belum ada kontainer. Tambahkan kontainer pertama untuk mulai mengisi identitas barang.";
      containerCardList.append(empty);
      return;
    }
    containerEntries.forEach((entry, containerIndex) => {
      const card = document.createElement("section");
      card.className = "initial-container-card";
      const header = document.createElement("header");
      const copy = document.createElement("div");
      const title = document.createElement("strong");
      title.textContent = `${entry.number} · ${entry.size === "40HC" ? "40' HC" : entry.size === "45HC" || entry.size === "45" ? "45' HC" : `${entry.size}'`}`;
      const small = document.createElement("small");
      small.textContent = `${entry.goods.length} identitas barang · dihitung satu kontainer pada YOR`;
      copy.append(title, small);
      const removeContainer = document.createElement("button");
      removeContainer.type = "button";
      removeContainer.className = "remove-line-button";
      removeContainer.textContent = "Hapus kontainer";
      removeContainer.addEventListener("click", () => {
        containerEntries.splice(containerIndex, 1);
        renderContainerEntries();
      });
      header.append(copy, removeContainer);
      const lines = document.createElement("div");
      lines.className = "initial-container-goods";
      entry.goods.forEach((goods, goodsIndex) => {
        const row = buildGoodsLine(goods, goodsIndex, syncInitialGoodsJSON, () => {
          entry.goods.splice(goodsIndex, 1);
          renderContainerEntries();
        }, entry.goods.length > 1);
        if (row) lines.append(row);
      });
      const addGoods = document.createElement("button");
      addGoods.type = "button";
      addGoods.className = "button secondary compact";
      addGoods.textContent = "+ Tambah identitas barang dalam kontainer";
      addGoods.addEventListener("click", () => {
        entry.goods.push(emptyGoodsLine());
        renderContainerEntries();
        const cards = $$('[data-initial-goods-line]', card);
        cards[cards.length - 1]?.scrollIntoView({ behavior: "smooth", block: "nearest" });
      });
      card.append(header, lines, addGoods);
      containerCardList.append(card);
    });
  }

  function renderLCLGoods() {
    syncInitialGoodsJSON();
    if (!lclGoodsLines) return;
    lclGoodsLines.replaceChildren();
    lclGoods.forEach((goods, index) => {
      const row = buildGoodsLine(goods, index, syncInitialGoodsJSON, () => {
        lclGoods.splice(index, 1);
        renderLCLGoods();
      }, lclGoods.length > 1);
      if (row) lclGoodsLines.append(row);
    });
  }

  function resetContainerEntries() {
    containerEntries = [];
    lclGoods = [emptyGoodsLine()];
    if (containerNumberDraft) containerNumberDraft.value = "";
    if (containerSizeDraft) containerSizeDraft.value = "";
    renderContainerEntries();
    renderLCLGoods();
  }

  function addContainerEntry() {
    const number = normalizeContainerDraft(containerNumberDraft?.value);
    const size = containerSizeDraft?.value || "";
    if (!number) {
      containerNumberDraft?.setCustomValidity("Gunakan 4 huruf dan 7 angka, contoh ABCD 123456-7.");
      containerNumberDraft?.reportValidity();
      return false;
    }
    containerNumberDraft?.setCustomValidity("");
    if (!size) {
      containerSizeDraft?.setCustomValidity("Pilih ukuran kontainer.");
      containerSizeDraft?.reportValidity();
      return false;
    }
    containerSizeDraft?.setCustomValidity("");
    if (containerEntries.some((entry) => entry.number === number)) {
      containerNumberDraft?.setCustomValidity("Nomor kontainer sudah ada dalam daftar.");
      containerNumberDraft?.reportValidity();
      return false;
    }
    containerEntries.push({ number, size, goods: [emptyGoodsLine()] });
    if (containerNumberDraft) containerNumberDraft.value = "";
    if (containerSizeDraft) containerSizeDraft.value = "";
    renderContainerEntries();
    containerNumberDraft?.focus();
    return true;
  }

  function updateLoadTypeFields() {
    const loadType = loadTypeSelect?.value || "";
    const isFCL = loadType === "FCL";
    const isLCL = loadType === "LCL";
    if (fclSection) fclSection.hidden = !isFCL;
    if (lclSection) lclSection.hidden = !isLCL;
    if (containersJSON) containersJSON.disabled = !isFCL;
    if (lclGoodsJSON) lclGoodsJSON.disabled = !isLCL;
    if (containerNumberDraft) containerNumberDraft.disabled = !isFCL;
    if (containerSizeDraft) containerSizeDraft.disabled = !isFCL;
    if (addContainerButton) addContainerButton.disabled = !isFCL;
    $$('input, select, textarea, button', containerCardList).forEach((field) => { field.disabled = !isFCL; });
    $$('input, select, textarea, button', lclGoodsLines).forEach((field) => { field.disabled = !isLCL; });
    if (addLCLGoodsButton) addLCLGoodsButton.disabled = !isLCL;
    if (lclVolumeInput) {
      lclVolumeInput.disabled = !isLCL;
      lclVolumeInput.required = isLCL;
      if (!isLCL) lclVolumeInput.value = "";
    }
  }

  loadTypeSelect?.addEventListener("change", updateLoadTypeFields);
  addContainerButton?.addEventListener("click", addContainerEntry);
  addLCLGoodsButton?.addEventListener("click", () => {
    lclGoods.push(emptyGoodsLine());
    renderLCLGoods();
    updateLoadTypeFields();
  });
  containerNumberDraft?.addEventListener("input", () => containerNumberDraft.setCustomValidity(""));
  containerSizeDraft?.addEventListener("change", () => containerSizeDraft.setCustomValidity(""));
  containerNumberDraft?.addEventListener("keydown", (event) => {
    if (event.key !== "Enter") return;
    event.preventDefault();
    addContainerEntry();
  });
  inventoryManualForm?.addEventListener("submit", (event) => {
    const loadType = loadTypeSelect?.value;
    if (loadType === "FCL") {
      if (containerNumberDraft?.value.trim()) addContainerEntry();
      const invalid = !containerEntries.length || containerEntries.some((entry) => !entry.goods.length || entry.goods.some((goods) => !goodsLineValid(goods)));
      if (invalid) {
        event.preventDefault();
        window.alert("Tambahkan minimal satu kontainer dan lengkapi seluruh identitas barang di setiap kontainer.");
        return;
      }
    }
    if (loadType === "LCL" && (!lclGoods.length || lclGoods.some((goods) => !goodsLineValid(goods)))) {
      event.preventDefault();
      window.alert("Lengkapi minimal satu identitas barang LCL beserta jenis, jumlah, dan satuannya.");
      return;
    }
    syncInitialGoodsJSON();
  });
  resetContainerEntries();
  updateLoadTypeFields();

  function updateInitialLocationChoice() {
    const selected = $('input[name="at_tpp"]:checked', createInventoryModal)?.value;
    const fields = $("[data-tpp-location-fields]", createInventoryModal);
    const select = $("[data-tpp-select]", createInventoryModal);
    const location = $("[data-tpp-location]", createInventoryModal);
    const atTPP = selected === "sudah";
    if (fields) fields.hidden = !atTPP;
    if (select) {
      select.disabled = !atTPP;
      select.required = atTPP;
      if (!atTPP) select.value = "";
    }
    if (location) {
      location.disabled = !atTPP;
      if (!atTPP) location.value = "";
    }
  }
  $$('[data-at-tpp-choice]', createInventoryModal || document).forEach((radio) => radio.addEventListener("change", updateInitialLocationChoice));
  updateInitialLocationChoice();

  function inventoryLoadSummary(item) {
    if (item?.container_no) return `${item.container_no}${item.container_size ? ` · ${item.container_size} kaki` : ""}`;
    const volume = Number(item?.estimated_volume_m3 || 0);
    return volume > 0 ? `LCL · ${new Intl.NumberFormat("id-ID", { maximumFractionDigits: 2 }).format(volume)} m³` : "LCL / tanpa kontainer";
  }

  const timelineModal = $("#timeline-modal");
  const timelineRoot = $("[data-timeline]", timelineModal);
  const timelineSummary = $("[data-timeline-summary]", timelineModal);
  const timelineTitle = $("#timeline-title", timelineModal);

  function renderTimeline(payload) {
    const item = payload.item || {};
    if (timelineTitle) timelineTitle.textContent = `Timeline ${item.determination_no || "barang"}`;
    if (timelineSummary) {
      timelineSummary.replaceChildren();
      const strong = document.createElement("strong");
      strong.textContent = `${inventoryLoadSummary(item)} · ${item.description || "Inventory"}`;
      const small = document.createElement("small");
      small.textContent = `${item.location_status || "Lokasi belum tercatat"} · Status: ${item.status_label || "—"}`;
      timelineSummary.append(strong, small);
    }
    if (!timelineRoot) return;
    timelineRoot.replaceChildren();
    const events = Array.isArray(payload.events) ? payload.events : [];
    if (!events.length) {
      const empty = document.createElement("div");
      empty.className = "empty-state small";
      empty.textContent = "Belum ada riwayat status.";
      timelineRoot.append(empty);
      return;
    }
    events.forEach((event) => {
      const row = document.createElement("article");
      row.className = "timeline-item";
      const marker = document.createElement("span");
      marker.className = "timeline-marker";
      const title = document.createElement("strong");
      title.textContent = event.label || event.code || "Pembaruan status";
      const detail = document.createElement("p");
      const documentDetail = event.document_no ? `${event.document_no}${event.document_date ? ` (${formatDate(event.document_date)})` : ""}` : "";
      detail.textContent = [documentDetail, event.notes].filter(Boolean).join(" · ") || "Status tersimpan dalam sistem";
      const meta = document.createElement("small");
      meta.textContent = `${formatDateTime(event.created_at)} · ${event.actor || "Sistem"}`;
      row.append(marker, title, detail, meta);
      const attachments = Array.isArray(event.attachments) ? event.attachments : [];
      if (attachments.length) {
        const attachmentList = document.createElement("div");
        attachmentList.className = "timeline-attachments";
        attachments.forEach((attachment) => {
          if (!attachment.download_url) return;
          const link = document.createElement("a");
          link.href = attachment.download_url;
          link.className = "timeline-attachment-link";
          link.textContent = `Unduh ${attachment.file_name || "dokumen"}`;
          link.setAttribute("download", "");
          attachmentList.append(link);
        });
        if (attachmentList.childElementCount) row.append(attachmentList);
      }
      timelineRoot.append(row);
    });
  }

  $$('[data-timeline-url]').forEach((button) => {
    button.addEventListener("click", async () => {
      if (timelineRoot) timelineRoot.innerHTML = '<div class="loading-state"><span class="spinner"></span>Memuat timeline…</div>';
      if (timelineSummary) timelineSummary.replaceChildren();
      openModal(timelineModal);
      try {
        const response = await fetch(button.dataset.timelineUrl, { headers: { Accept: "application/json" } });
        if (!response.ok) throw new Error("Timeline belum dapat dimuat");
        renderTimeline(await response.json());
      } catch (error) {
        if (timelineRoot) {
          timelineRoot.replaceChildren();
          const message = document.createElement("div");
          message.className = "empty-state small";
          message.textContent = error.message;
          timelineRoot.append(message);
        }
      }
    });
  });

  const detailModal = $("#inventory-detail-modal");
  const detailContent = $("[data-detail-content]", detailModal);

  function detailField(label, value, wide = false) {
    const node = document.createElement("div");
    node.className = wide ? "detail-field wide" : "detail-field";
    const name = document.createElement("span");
    name.textContent = label;
    const content = document.createElement("strong");
    content.textContent = value || "—";
    node.append(name, content);
    return node;
  }

  function inventoryBlockLocation(item) {
    if (!item?.at_tpp) return "";
    const location = String(item.location || "").trim();
    const facility = String(item.facility_name || "").trim();
    const status = String(item.location_status || "").trim();
    if (!location) return "";
    const normalized = location.toLocaleLowerCase("id-ID");
    if ((facility && normalized === facility.toLocaleLowerCase("id-ID")) || (status && normalized === status.toLocaleLowerCase("id-ID"))) return "";
    return location;
  }

  function renderInventoryDetail(item) {
    if (!detailModal || !detailContent) return;
    const isTitipan = item.item_type === "TITIPAN";
    const typeLabel = isTitipan ? "Barang Titipan" : (item.item_type || "—");
    $("[data-detail-title]", detailModal).textContent = item.determination_no || (isTitipan ? "Informasi pemasukan" : "Informasi penetapan");
    $("[data-detail-subtitle]", detailModal).textContent = `${typeLabel} · ${item.status_label || "—"}`;
    detailContent.replaceChildren();
    const grid = document.createElement("div");
    grid.className = "detail-grid";
    const lartas = item.is_restricted ? `Ya${item.restriction_rule ? ` — ${item.restriction_rule}` : ""}` : "Tidak";
    const exitLabels = { impor_untuk_dipakai: "IMPOR UTK DIPAKAI", reekspor: "REEKSPOR", batal_ekspor: "BATAL EKSPOR", ekspor: "EKSPOR", keluarkan_ke_tpb: "KELUARKAN KE TPB", lelang: "LELANG", musnah: "MUSNAH", psp: "PSP", hibah: "HIBAH", bmmn: "BMMN", diserahkan_ke_aph_lain: "DISERAHKAN KE APH LAIN", pembatalan_bdn: "PEMBATALAN BDN", diserahkan_ke_ppns: "DISERAHKAN KE PPNS", penghapusan: "PENGHAPUSAN", pengeluaran_barang_titipan: "PENGELUARAN BARANG TITIPAN" };
    const fields = [
      detailField("ID inventory", item.id),
      detailField("Referensi barang", item.reference_no),
      detailField(isTitipan ? "Nomor dokumen dasar pemasukan" : "Nomor penetapan", item.determination_no),
      detailField(isTitipan ? "Tanggal dokumen" : "Tanggal penetapan", formatDate(item.determination_date)),
      detailField("Jenis inventory / asal", `${typeLabel} / ${item.origin_type || "—"}`),
      detailField("Status inventory", item.is_active ? "Aktif" : "Selesai / tidak aktif"),
    ];
    if (isTitipan) {
      fields.push(
        detailField("Kategori barang titipan", item.entrusted_category),
        detailField("Kantor/unit penitip", item.source_office),
      );
    }
    fields.push(
      ...((item.item_type === "BTD" || item.bl_no || item.bl_date) ? [detailField("Nomor BL", item.bl_no), detailField("Tanggal BL", formatDate(item.bl_date))] : []),
      detailField("Nomor manifest / pos", [item.manifest_no, item.manifest_position].filter(Boolean).join(" / ")),
      detailField("Tanggal manifest", formatDate(item.manifest_date)),
      detailField("Jenis muatan", item.load_type || "—"),
      detailField(item.container_no ? "Kontainer & ukuran" : "Perkiraan volume LCL", inventoryLoadSummary(item)),
      detailField("ID unit fisik", item.physical_unit_id),
      detailField("Penghitung utama kapasitas", item.occupancy_primary ? "Ya" : "Tidak"),
      detailField("Uraian barang", item.description, true),
      detailField("Jenis barang", item.item_kind),
      detailField("Kondisi barang", item.goods_condition),
      detailField("Jumlah", `${item.quantity || 0} ${item.unit || ""}`.trim()),
      detailField("Detail jumlah barang", item.quantity_detail, true),
      detailField("Nilai barang", formatMoney(item.goods_value)),
      detailField("Status lokasi", item.location_status),
      detailField("Blok TPP", inventoryBlockLocation(item)),
      ...(!item.at_tpp ? [detailField("Lokasi fisik / TPS", item.location)] : []),
    );
    if (!isTitipan) fields.push(detailField("TPS asal", item.origin_warehouse));
    fields.push(
      detailField("TPP", item.facility_name || "Belum berada di TPP"),
      detailField("ID TPP", item.facility_id),
      detailField("Pemilik", item.owner_name),
      detailField("Alamat pemilik", item.owner_address, true),
      detailField("Request Penelitian PFPD", item.research_request_no),
      detailField("Tanggal request", formatDate(item.research_request_date)),
      detailField("Wajib penelitian PFPD", item.pfpd_required ? "Ya" : "Tidak"),
      detailField("Kode HS", item.hs_code),
      detailField("Lartas", lartas),
      detailField("Proses penyelesaian aktif", item.current_disposition ? item.current_disposition.toUpperCase() : "Tidak ada"),
      detailField("Kode status", item.status_code),
      detailField("Terakhir diperbarui", formatDateTime(item.updated_at)),
    );
    if (item.category) fields.push(detailField("Kategori BDN", item.category, true));
    if (item.item_type === "BMMN") {
      fields.push(
        detailField("Jenis dokumen asal BMMN", item.origin_document_type),
        detailField("Nomor dokumen asal", item.origin_document_no),
        detailField("Tanggal dokumen asal", formatDate(item.origin_document_date)),
        detailField("Usulan peruntukan", item.allocation_proposal_type),
        detailField("Dokumen usulan", item.allocation_proposal_no ? `${item.allocation_proposal_no} · ${formatDate(item.allocation_proposal_date)}` : "—"),
        detailField("Persetujuan peruntukan", item.allocation_approval_type),
        detailField("Peruntukan BMMN saat ini", item.allocation_purpose),
        detailField("Dokumen persetujuan", item.allocation_approval_no ? `${item.allocation_approval_no} · ${formatDate(item.allocation_approval_date)}` : "—"),
      );
    }
    fields.push(
      detailField("Dokumen pengeluaran", item.exit_document_no ? `${item.exit_document_no} · ${formatDate(item.exit_document_date)}` : "—"),
      detailField("Jenis pengeluaran", exitLabels[item.exit_type] || "—"),
      detailField("Keterangan pengeluaran", item.exit_notes, true),
      detailField("Dibuat oleh", item.created_by),
      detailField("Tanggal dibuat", formatDateTime(item.created_at)),
    );
    fields.forEach((field) => grid.append(field));
    detailContent.append(grid);
  }

  $$('[data-detail-url]').forEach((button) => {
    button.addEventListener("click", async () => {
      if (detailContent) detailContent.innerHTML = '<div class="loading-state"><span class="spinner"></span>Memuat detail…</div>';
      openModal(detailModal);
      try {
        const response = await fetch(button.dataset.detailUrl, { headers: { Accept: "application/json" } });
        if (!response.ok) throw new Error("Detail belum dapat dimuat");
        const payload = await response.json();
        renderInventoryDetail(payload.item || {});
      } catch (error) {
        if (detailContent) detailContent.textContent = error.message;
      }
    });
  });

  function updateActionModalHeader(form, step = null) {
    const modal = form?.closest(".action-modal");
    if (!modal) return;
    const title = $("[data-action-modal-title]", modal);
    const description = $("[data-action-modal-description]", modal);
    if (title) title.textContent = step?.dataset.stepLabel || title.dataset.defaultTitle || "Pilih submenu action";
    if (description) {
      description.textContent = step
        ? `${step.dataset.stepDescription || "Lengkapi data action."} Pilih barang dan isi dokumen pada form di bawah.`
        : description.dataset.defaultDescription || "Pilih action yang ingin dikerjakan.";
    }
  }

  function resetActionForm(form) {
    if (!form) return;
    form.reset();
    $$('[data-step-code]', form).forEach((step) => step.classList.remove("selected"));
    const code = $("[data-event-code]", form);
    if (code) code.value = "";
    const selected = $("[data-selected-step]", form);
    if (selected) selected.hidden = true;
    const menu = $("[data-action-menu]", form);
    const detail = $("[data-action-detail]", form);
    if (menu) menu.hidden = false;
    if (detail) detail.hidden = true;
    $$('[data-action-fields], [data-process-action-fields], [data-inventory-picker], [data-process-picker], [data-pfpd-request-picker], [data-relocation-picker], [data-auction-schedule-picker]', form).forEach((section) => { section.hidden = true; });
    $$('[data-fields-for], [data-process-fields-for]', form).forEach((section) => {
      section.hidden = true;
      $$('input, select, textarea', section).forEach((field) => { field.disabled = true; });
    });
    $$('input[name="document_no"], input[name="document_date"], input[name="document_file"]', form).forEach((field) => { field.disabled = true; });
    $$('[data-picker-checkbox], [data-process-candidate-checkbox]', form).forEach((box) => {
      box.checked = false;
      box.disabled = true;
    });
    $$('[data-pfpd-request-card]', form).forEach((card) => {
      card.hidden = false;
      card.classList.remove('active');
      const items = $('[data-pfpd-request-items]', card);
      if (items) items.hidden = true;
      $$('input, select, textarea', card).forEach((field) => { field.disabled = true; });
    });
    const pfpdEmpty = $('[data-pfpd-search-empty]', form);
    if (pfpdEmpty) pfpdEmpty.hidden = true;
    const submit = $("[data-submit-step]", form);
    if (submit) {
      submit.disabled = true;
      submit.hidden = true;
    }
    updateActionModalHeader(form);
    const scroll = $(".action-modal-scroll", form);
    if (scroll) scroll.scrollTop = 0;
  }

  function showActionDetail(form, step) {
    const menu = $("[data-action-menu]", form);
    const detail = $("[data-action-detail]", form);
    const submit = $("[data-submit-step]", form);
    if (menu) menu.hidden = true;
    if (detail) detail.hidden = false;
    if (submit) submit.hidden = false;
    updateActionModalHeader(form, step);
    const scroll = $(".action-modal-scroll", form);
    if (scroll) scroll.scrollTop = 0;
  }

  function selectStep(form, step, callback) {
    $$('[data-step-code]', form).forEach((other) => other.classList.remove("selected"));
    step.classList.add("selected");
    const code = $("[data-event-code]", form);
    if (code) code.value = step.dataset.stepCode || "";
    const selected = $("[data-selected-step]", form);
    const selectedLabel = $("[data-selected-step-label]", form);
    if (selected) selected.hidden = false;
    if (selectedLabel) selectedLabel.textContent = step.dataset.stepLabel || "";
    const documentLabel = $("[data-document-label]", form);
    if (documentLabel?.firstChild) documentLabel.firstChild.nodeValue = `${step.dataset.stepDocument || "Nomor dokumen"} `;
    showActionDetail(form, step);
    callback(step);
  }

  function bindStepPicker(form, callback) {
    if (!form) return;
    $$('[data-step-code]', form).forEach((step) => step.addEventListener("click", () => selectStep(form, step, callback)));
  }

  const inventoryDrawer = $("#inventory-action-drawer");
  const inventoryForm = $("[data-inventory-action-form]", inventoryDrawer);
  const inventoryPickerItems = $$('[data-picker-item]', inventoryForm);
  const inventorySearch = $("[data-inventory-picker-search]", inventoryForm);
  const inventorySelectAll = $("[data-picker-select-all]", inventoryForm);
  const pfpdRequestPicker = $("[data-pfpd-request-picker]", inventoryForm);
  const pfpdRequestCards = $$('[data-pfpd-request-card]', inventoryForm);
  const pfpdRequestSearch = $('[data-pfpd-request-search]', inventoryForm);
  const pfpdSearchEmpty = $('[data-pfpd-search-empty]', inventoryForm);
  const pfpdResultsJSON = $('[data-pfpd-results-json]', inventoryForm);
  const censusTargetPicker = $('[data-census-target-picker]', inventoryForm);
  const censusTargetCards = $$('[data-census-target-card]', inventoryForm);
  const censusTargetSearch = $('[data-census-target-search]', inventoryForm);
  const censusTargetEmpty = $('[data-census-target-empty]', inventoryForm);
  const censusResultsJSON = $('[data-census-results-json]', inventoryForm);
  const censusResultTemplate = $('[data-census-result-line-template]', inventoryForm);
  const relocationPicker = $('[data-relocation-picker]', inventoryForm);
  const relocationSourceList = $('[data-relocation-source-list]', inventoryForm);
  const relocationSources = $$('[data-relocation-source]', inventoryForm);
  const relocationSearch = $('[data-relocation-search]', inventoryForm);
  const relocationMode = $('[data-relocation-mode]', inventoryForm);
  const relocationEmpty = $('[data-relocation-empty]', inventoryForm);
  const relocationJSON = $('[data-relocation-json]', inventoryForm);
  const relocationEditorList = $('[data-relocation-editor-list]', inventoryForm);
  const relocationOperationTemplate = $('[data-relocation-operation-template]', inventoryForm);
  const relocationItemTemplate = $('[data-relocation-item-template]', inventoryForm);
  const relocationAllocationTemplate = $('[data-relocation-allocation-template]', inventoryForm);
  const inventoryPickerList = $('[data-inventory-picker-list]', inventoryForm);
  const censusTargetList = $('.census-target-list', inventoryForm);
  const pfpdRequestList = $('.pfpd-request-list', inventoryForm);
  let inventoryStep = null;
  let activePFPDRequestCard = null;

  function reorderSelectionGroup(container, items, selectedPredicate, activeItem = null) {
    if (!container || !items?.length) return;
    const selected = [];
    const active = [];
    const others = [];
    const hidden = [];
    items.forEach((item) => {
      if (item.hidden) {
        hidden.push(item);
        return;
      }
      if (activeItem && item === activeItem) {
        active.push(item);
        return;
      }
      if (selectedPredicate(item)) selected.push(item);
      else others.push(item);
    });
    [...active, ...selected, ...others, ...hidden].forEach((item) => container.appendChild(item));
  }

  function selectedInventoryPickerItems() {
    return inventoryPickerItems.filter((item) => $('[data-picker-checkbox]', item)?.checked);
  }

  function selectedCensusTargetCards() {
    return censusTargetCards.filter((card) => $('[data-census-target-checkbox]', card)?.checked);
  }

  function relocationSourceItems(source) {
    return $$('[data-relocation-source-item]', source);
  }

  function selectedRelocationSources() {
    return relocationSources.filter((source) => $('[data-relocation-source-checkbox]', source)?.checked);
  }

  function relocationSourceEligible(source) {
    if (!source) return false;
    const items = relocationSourceItems(source);
    return items.length > 0 && items.every((item) => Number(item.dataset.quantity || 0) > 0);
  }

  function relocationSourceMatchesMode(source) {
    const mode = relocationMode?.value || 'bongkar';
    const loadType = (source?.dataset.loadType || '').toUpperCase();
    if (mode === 'muat') return loadType === 'LCL';
    return loadType === 'FCL';
  }

  function filterRelocationSources() {
    const term = (relocationSearch?.value || '').trim().toLowerCase();
    let visible = 0;
    relocationSources.forEach((source) => {
      const eligible = relocationSourceEligible(source) && relocationSourceMatchesMode(source);
      const matches = !term || (source.dataset.search || '').includes(term);
      source.hidden = !(eligible && matches);
      const checkbox = $('[data-relocation-source-checkbox]', source);
      if (checkbox) {
        checkbox.disabled = !(eligible && inventoryStep?.dataset.stepCode === 'pindah_bongkar_kontainer');
        if (!eligible && checkbox.checked) checkbox.checked = false;
      }
      source.classList.toggle('selected', checkbox?.checked || false);
      if (!source.hidden) visible++;
    });
    if (relocationEmpty) relocationEmpty.hidden = visible > 0 || relocationSources.length === 0;
    reorderSelectionGroup(relocationSourceList, relocationSources, (source) => $('[data-relocation-source-checkbox]', source)?.checked);
    updateRelocationSelection();
  }

  function getRelocationSourceByKey(key) {
    return relocationSources.find((source) => source.dataset.sourceKey === key) || null;
  }

  function getRelocationItemByID(id) {
    for (const source of relocationSources) {
      const item = relocationSourceItems(source).find((candidate) => candidate.dataset.inventoryId === id);
      if (item) return item;
    }
    return null;
  }

  function relocationOperationCards() {
    return $$('[data-relocation-operation]', relocationEditorList);
  }

  function relocationItemCards(container = relocationEditorList) {
    return $$('[data-relocation-item-operation]', container);
  }

  function relocationRows(itemCard) {
    return $$('[data-relocation-allocation-row]', $('[data-relocation-allocation-list]', itemCard));
  }

  function setRelocationRowMode(row, mode = 'bongkar') {
    if (!row) return;
    const loadTypeField = $('[data-relocation-field="load_type"]', row);
    if (!loadTypeField) return;
    const lclOption = $('option[value="LCL"]', loadTypeField);
    if (mode === 'muat') {
      loadTypeField.value = 'FCL';
      if (lclOption) lclOption.disabled = true;
    } else if (lclOption) {
      lclOption.disabled = false;
    }
    const type = loadTypeField.value || 'FCL';
    const isFCL = type === 'FCL';
    $$('[data-relocation-container-field]', row).forEach((field) => { field.hidden = !isFCL; });
    const volumeField = $('[data-relocation-volume-field]', row);
    if (volumeField) volumeField.hidden = isFCL;
    const containerNo = $('[data-relocation-field="container_no"]', row);
    const containerSize = $('[data-relocation-field="container_size"]', row);
    const volume = $('[data-relocation-field="estimated_volume_m3"]', row);
    if (containerNo) {
      containerNo.disabled = !isFCL;
      containerNo.required = isFCL;
      if (!isFCL) containerNo.value = '';
    }
    if (containerSize) {
      containerSize.disabled = !isFCL;
      containerSize.required = isFCL;
    }
    if (volume) {
      volume.disabled = isFCL;
      volume.required = !isFCL;
      if (isFCL) volume.value = '';
    }
  }

  function relocationItemHasChange(itemCard, rows = relocationRows(itemCard)) {
    const sourceItem = getRelocationItemByID(itemCard?.dataset.inventoryId || '');
    if (!sourceItem || !rows.length) return false;
    if (rows.length > 1) return true;
    const row = rows[0];
    const destinationType = $('[data-relocation-field="load_type"]', row)?.value || '';
    const sourceType = sourceItem.dataset.loadType || '';
    if (destinationType !== sourceType) return true;
    if (destinationType === 'FCL') {
      const compact = (value) => String(value || '').replace(/[^a-z0-9]/gi, '').toUpperCase();
      const destinationContainer = compact($('[data-relocation-field="container_no"]', row)?.value || '');
      const destinationSize = $('[data-relocation-field="container_size"]', row)?.value || '';
      return destinationContainer !== compact(sourceItem.dataset.containerNo || '') || destinationSize !== (sourceItem.dataset.containerSize || '');
    }
    if (destinationType === 'LCL') {
      const destinationVolume = Number($('[data-relocation-field="estimated_volume_m3"]', row)?.value || 0);
      return Math.abs(destinationVolume - Number(sourceItem.dataset.volume || 0)) >= 0.005;
    }
    return false;
  }

  function updateRelocationItem(itemCard) {
    if (!itemCard) return false;
    const sourceItem = getRelocationItemByID(itemCard.dataset.inventoryId || '');
    if (!sourceItem) return false;
    const mode = relocationMode?.value || 'bongkar';
    const rows = relocationRows(itemCard);
    rows.forEach((row, index) => {
      const title = $('[data-relocation-row-title]', row);
      if (title) title.textContent = `Tujuan ${index + 1}`;
      setRelocationRowMode(row, mode);
    });
    const quantity = Number(sourceItem.dataset.quantity || 0);
    const unit = sourceItem.dataset.unit || '';
    const sourceTitle = $('[data-relocation-source-title]', itemCard);
    const sourceDescription = $('[data-relocation-source-description]', itemCard);
    const required = $('[data-relocation-required-quantity]', itemCard);
    const modeCopy = $('[data-relocation-mode-copy]', itemCard);
    const destinationTitle = $('[data-relocation-destination-title]', itemCard);
    const destinationHelp = $('[data-relocation-destination-help]', itemCard);
    if (sourceTitle) sourceTitle.textContent = `${sourceItem.dataset.determinationNo || 'Dokumen'} · ${new Intl.NumberFormat('id-ID', { maximumFractionDigits: 2 }).format(quantity)} ${unit}`;
    if (sourceDescription) sourceDescription.textContent = sourceItem.dataset.description || '—';
    if (required) required.textContent = `${new Intl.NumberFormat('id-ID', { maximumFractionDigits: 2 }).format(quantity)} ${unit}`.trim();
    if (modeCopy) modeCopy.textContent = mode === 'muat' ? 'Seluruh kuantitas uraian ini harus dimuat ke kontainer tujuan.' : 'Jumlah seluruh tujuan harus sama persis dengan kuantitas uraian ini.';
    if (destinationTitle) destinationTitle.textContent = mode === 'muat' ? 'Tujuan muat ke kontainer' : 'Tujuan bongkar / pindah';
    if (destinationHelp) destinationHelp.textContent = mode === 'muat'
      ? 'Tambahkan satu atau beberapa kontainer tujuan untuk uraian LCL ini.'
      : 'Pilih FCL untuk pindah kontainer atau LCL untuk dibongkar ke gudang.';

    const count = $('[data-relocation-allocation-count]', itemCard);
    if (count) count.textContent = `${rows.length} tujuan`;
    const total = rows.reduce((sum, row) => sum + Number($('[data-relocation-field="quantity"]', row)?.value || 0), 0);
    const totalLabel = $('[data-relocation-total]', itemCard);
    if (totalLabel) totalLabel.textContent = `${new Intl.NumberFormat('id-ID', { maximumFractionDigits: 2 }).format(total)} ${unit}`.trim();
    const balance = $('[data-relocation-balance]', itemCard);
    const bar = $('[data-relocation-total-bar]', itemCard);
    const exact = rows.length > 0 && Math.abs(total - quantity) < 0.005;
    const changed = relocationItemHasChange(itemCard, rows);
    const valid = Boolean(exact && changed && rows.every((row) => {
      const allocationType = $('[data-relocation-field="load_type"]', row)?.value || '';
      const qty = Number($('[data-relocation-field="quantity"]', row)?.value || 0);
      if (!(qty > 0)) return false;
      if (allocationType === 'FCL') return Boolean($('[data-relocation-field="container_no"]', row)?.value.trim() && $('[data-relocation-field="container_size"]', row)?.value);
      if (allocationType === 'LCL') return Number($('[data-relocation-field="estimated_volume_m3"]', row)?.value || 0) > 0;
      return false;
    }));
    if (balance) {
      if (!exact) balance.textContent = `Selisih ${new Intl.NumberFormat('id-ID', { maximumFractionDigits: 2 }).format(quantity - total)} ${unit}`.trim();
      else balance.textContent = changed ? 'Sesuai dengan kuantitas sumber' : 'Ubah tujuan atau tambahkan pembagian';
    }
    if (bar) bar.classList.toggle('valid', valid);
    itemCard.dataset.valid = valid ? 'true' : 'false';
    return valid;
  }

  function updateRelocationGroup(groupCard) {
    if (!groupCard) return false;
    const itemCards = relocationItemCards(groupCard);
    const valid = itemCards.length > 0 && itemCards.every((itemCard) => updateRelocationItem(itemCard));
    groupCard.dataset.valid = valid ? 'true' : 'false';
    return valid;
  }

  function buildRelocationDefaultAllocation(sourceItem) {
    const mode = relocationMode?.value || 'bongkar';
    const quantity = Number(sourceItem?.dataset.quantity || 0);
    if (mode === 'muat') {
      return { load_type: 'FCL', container_no: '', container_size: '20', estimated_volume_m3: 0, quantity };
    }
    return {
      load_type: sourceItem?.dataset.loadType || 'FCL',
      container_no: sourceItem?.dataset.containerNo || '',
      container_size: sourceItem?.dataset.containerSize || '20',
      estimated_volume_m3: Number(sourceItem?.dataset.volume || 0),
      quantity,
    };
  }

  function addRelocationAllocation(itemCard, values = {}) {
    if (!relocationAllocationTemplate || !itemCard) return;
    const fragment = relocationAllocationTemplate.content.cloneNode(true);
    const row = fragment.querySelector('[data-relocation-allocation-row]');
    const list = $('[data-relocation-allocation-list]', itemCard);
    if (!row || !list) return;
    list.append(row);
    const loadType = $('[data-relocation-field="load_type"]', row);
    const quantity = $('[data-relocation-field="quantity"]', row);
    const containerNo = $('[data-relocation-field="container_no"]', row);
    const containerSize = $('[data-relocation-field="container_size"]', row);
    const volume = $('[data-relocation-field="estimated_volume_m3"]', row);
    if (loadType) loadType.value = values.load_type || 'FCL';
    if (quantity && values.quantity !== undefined) quantity.value = values.quantity;
    if (containerNo) containerNo.value = values.container_no || '';
    if (containerSize) containerSize.value = values.container_size || '';
    if (volume && values.estimated_volume_m3) volume.value = values.estimated_volume_m3;
    setRelocationRowMode(row, relocationMode?.value || 'bongkar');
    loadType?.addEventListener('change', () => {
      updateRelocationItem(itemCard);
      refreshRelocationValidity();
    });
    $$('input, select', row).forEach((field) => field.addEventListener('input', () => {
      // Jangan menjalankan sinkronisasi/reorder DOM saat pengguna mengetik.
      // Memindahkan ulang ancestor dari input aktif membuat browser melepas fokus,
      // sehingga sebelumnya kursor keluar setelah setiap karakter.
      updateRelocationItem(itemCard);
      refreshRelocationValidity();
    }));
    $('[data-remove-relocation-allocation]', row)?.addEventListener('click', () => {
      row.remove();
      updateRelocationItem(itemCard);
      refreshRelocationValidity();
    });
    updateRelocationItem(itemCard);
    refreshRelocationValidity();
  }

  function ensureRelocationOperationCard(source) {
    if (!relocationOperationTemplate || !relocationItemTemplate || !relocationEditorList || !source) return null;
    const sourceKey = source.dataset.sourceKey || '';
    const existing = relocationOperationCards().find((card) => card.dataset.sourceKey === sourceKey);
    if (existing) return existing;
    const fragment = relocationOperationTemplate.content.cloneNode(true);
    const groupCard = fragment.querySelector('[data-relocation-operation]');
    if (!groupCard) return null;
    groupCard.dataset.sourceKey = sourceKey;
    const title = $('[data-relocation-group-title]', groupCard);
    const subtitle = $('[data-relocation-group-subtitle]', groupCard);
    const count = $('[data-relocation-group-count]', groupCard);
    const items = relocationSourceItems(source);
    const isFCL = (source.dataset.loadType || '').toUpperCase() === 'FCL';
    if (title) title.textContent = isFCL
      ? `${source.dataset.containerNo || 'Kontainer'} · ${source.dataset.containerSize || '—'}`
      : `LCL · ${source.dataset.determinationNo || 'Dokumen'}`;
    if (subtitle) subtitle.textContent = isFCL
      ? 'Seluruh uraian dalam kontainer ini harus memperoleh tujuan.'
      : 'Uraian LCL diproses sebagai satu target mandiri.';
    if (count) count.textContent = `${items.length} uraian`;
    const itemList = $('[data-relocation-item-editor-list]', groupCard);
    items.forEach((sourceItem, index) => {
      const itemFragment = relocationItemTemplate.content.cloneNode(true);
      const itemCard = itemFragment.querySelector('[data-relocation-item-operation]');
      if (!itemCard || !itemList) return;
      itemCard.dataset.inventoryId = sourceItem.dataset.inventoryId || '';
      itemCard.dataset.sourceKey = sourceKey;
      itemCard.dataset.itemIndex = String(index + 1);
      itemList.append(itemCard);
      $('[data-add-relocation-allocation]', itemCard)?.addEventListener('click', () => addRelocationAllocation(itemCard));
      addRelocationAllocation(itemCard, buildRelocationDefaultAllocation(sourceItem));
    });
    relocationEditorList.append(groupCard);
    updateRelocationGroup(groupCard);
    return groupCard;
  }

  function syncRelocationEditors() {
    const selected = selectedRelocationSources();
    if (relocationEditorList) relocationEditorList.hidden = selected.length === 0;
    const selectedKeys = new Set(selected.map((source) => source.dataset.sourceKey || ''));
    relocationOperationCards().forEach((card) => {
      if (!selectedKeys.has(card.dataset.sourceKey || '')) card.remove();
    });
    selected.forEach((source) => ensureRelocationOperationCard(source));
    const cards = relocationOperationCards();
    reorderSelectionGroup(relocationEditorList, cards, (card) => selectedKeys.has(card.dataset.sourceKey || ''));
    cards.forEach((card) => updateRelocationGroup(card));
  }

  function refreshRelocationValidity() {
    const selected = selectedRelocationSources();
    const selectedKeys = new Set(selected.map((source) => source.dataset.sourceKey || ''));
    const cards = relocationOperationCards().filter((card) => selectedKeys.has(card.dataset.sourceKey || ''));
    const ready = selected.length > 0 && cards.length === selected.length && cards.every((card) => updateRelocationGroup(card));
    const submit = $('[data-submit-step]', inventoryForm);
    if (submit && inventoryStep?.dataset.stepCode === 'pindah_bongkar_kontainer') submit.disabled = !ready;
    return ready;
  }

  function updateRelocationSelection() {
    relocationSources.forEach((source) => {
      const selected = $('[data-relocation-source-checkbox]', source)?.checked || false;
      source.classList.toggle('selected', selected);
    });
    const selected = selectedRelocationSources();
    const sourceCount = $('[data-relocation-source-count]', inventoryForm);
    if (sourceCount) sourceCount.textContent = `${selected.length} target dipilih`;
    reorderSelectionGroup(relocationSourceList, relocationSources, (source) => $('[data-relocation-source-checkbox]', source)?.checked);
    syncRelocationEditors();
    refreshRelocationValidity();
  }

  function serializeRelocation() {
    if (!relocationJSON) return false;
    const selected = selectedRelocationSources();
    const selectedKeys = new Set(selected.map((source) => source.dataset.sourceKey || ''));
    const groupCards = relocationOperationCards().filter((card) => selectedKeys.has(card.dataset.sourceKey || ''));
    const operations = groupCards.flatMap((groupCard) => relocationItemCards(groupCard).map((itemCard) => {
      const allocations = relocationRows(itemCard).map((row) => ({
        load_type: $('[data-relocation-field="load_type"]', row)?.value || '',
        container_no: $('[data-relocation-field="container_no"]', row)?.value.trim() || '',
        container_size: $('[data-relocation-field="container_size"]', row)?.value || '',
        estimated_volume_m3: Number($('[data-relocation-field="estimated_volume_m3"]', row)?.value || 0),
        quantity: Number($('[data-relocation-field="quantity"]', row)?.value || 0),
      }));
      return { inventory_id: itemCard.dataset.inventoryId || '', allocations };
    }));
    const expectedItemCount = selected.reduce((sum, source) => sum + relocationSourceItems(source).length, 0);
    const invalid = !selected.length || groupCards.length !== selected.length || operations.length !== expectedItemCount || groupCards.some((card) => card.dataset.valid !== 'true') || operations.some((operation) => {
      const sourceItem = getRelocationItemByID(operation.inventory_id);
      if (!sourceItem || !operation.allocations.length) return true;
      const total = operation.allocations.reduce((sum, allocation) => sum + allocation.quantity, 0);
      return Math.abs(total - Number(sourceItem.dataset.quantity || 0)) >= 0.005;
    });
    if (invalid) return false;
    relocationJSON.value = JSON.stringify({ mode: relocationMode?.value || 'bongkar', operations });
    relocationJSON.disabled = false;
    return true;
  }

  function setCensusTargetState(card) {
    const selected = $('[data-census-target-checkbox]', card)?.checked || false;
    const editor = $('[data-census-target-editor]', card);
    if (editor) editor.hidden = !selected;
    $$('input, select, textarea, button', editor).forEach((field) => {
      field.disabled = !selected;
      if (field.matches('[data-census-field]')) {
        field.required = selected && !['goods_value', 'quantity_detail'].includes(field.dataset.censusField);
      }
    });
    card.classList.toggle('selected', selected);
  }

  function updateCensusTargetSelection() {
    censusTargetCards.forEach(setCensusTargetState);
    const selected = selectedCensusTargetCards();
    const count = $('[data-census-picker-count]', inventoryForm);
    if (count) count.textContent = `${selected.length} target dipilih`;
    const submit = $('[data-submit-step]', inventoryForm);
    if (submit && inventoryStep?.dataset.stepCode === 'pencacahan') submit.disabled = selected.length === 0;
    reorderSelectionGroup(censusTargetList, censusTargetCards, (card) => $('[data-census-target-checkbox]', card)?.checked);
  }

  function filterCensusTargets() {
    const term = (censusTargetSearch?.value || '').trim().toLowerCase();
    let visible = 0;
    censusTargetCards.forEach((card) => {
      const matches = !term || (card.dataset.censusSearch || '').includes(term);
      card.hidden = !matches;
      if (matches) visible++;
    });
    if (censusTargetEmpty) censusTargetEmpty.hidden = visible > 0 || censusTargetCards.length === 0;
    reorderSelectionGroup(censusTargetList, censusTargetCards, (card) => $('[data-census-target-checkbox]', card)?.checked);
  }

  function addCensusTargetLine(card) {
    if (!card || card.dataset.loadType !== 'FCL' || !censusResultTemplate) return;
    const fragment = censusResultTemplate.content.cloneNode(true);
    const row = fragment.querySelector('[data-census-result-line]');
    const container = $('[data-census-target-lines]', card);
    if (!row || !container) return;
    container.append(row);
    $$('[data-census-field]', row).forEach((field) => {
      field.disabled = false;
      field.required = !['goods_value', 'quantity_detail'].includes(field.dataset.censusField);
    });
    $('[data-remove-census-target-line]', row)?.addEventListener('click', () => row.remove());
    $('[data-census-field="description"]', row)?.focus();
  }

  function serializeCensusResults() {
    if (!censusResultsJSON) return false;
    const targets = selectedCensusTargetCards().map((card) => ({
      target_id: card.dataset.targetId || '',
      load_type: card.dataset.loadType || '',
      lines: $$('[data-census-result-line]', card).map((row) => ({
        inventory_id: row.dataset.inventoryId || '',
        description: $('[data-census-field="description"]', row)?.value.trim() || '',
        item_kind: $('[data-census-field="item_kind"]', row)?.value || '',
        goods_value: $('[data-census-field="goods_value"]', row)?.value.trim() || '',
        quantity: Number($('[data-census-field="quantity"]', row)?.value || 0),
        quantity_detail: $('[data-census-field="quantity_detail"]', row)?.value.trim() || '',
        unit: $('[data-census-field="unit"]', row)?.value || '',
        goods_condition: $('[data-census-field="goods_condition"]', row)?.value || '',
      })),
    }));
    const invalid = !targets.length || targets.some((target) => !target.target_id || !target.lines.length || target.lines.some((line) => !line.description || !line.item_kind || !(line.quantity > 0) || !line.unit || !line.goods_condition));
    if (invalid) return false;
    censusResultsJSON.value = JSON.stringify(targets);
    censusResultsJSON.disabled = false;
    return true;
  }

  function inventoryItemEligible(item) {
    if (!inventoryStep) return false;
    const kind = item.dataset.itemType;
    const currentDisposition = item.dataset.currentDisposition || "";
    const status = item.dataset.itemStatus || "";
    const stepCode = inventoryStep.dataset.stepCode || "";
    const completedProcess = ["laku", "alokasi_hasil_lelang", "ba_musnah", "ba_serah_terima"].includes(status);
    if ((currentDisposition || completedProcess) && stepCode !== "pengeluaran_barang") return false;
    if (stepCode === "pengeluaran_barang" && currentDisposition) {
      const canExitAuction = currentDisposition === "lelang" && ["laku", "alokasi_hasil_lelang"].includes(status);
      const canExitDestruction = currentDisposition === "musnah" && ["kep_musnah", "ba_musnah"].includes(status);
      if (!canExitAuction && !canExitDestruction) return false;
    }
    if (inventoryStep.dataset.bmmnOnly === "true" && kind !== "BMMN") return false;
    if (inventoryStep.dataset.nonBmmnOnly === "true" && kind === "BMMN") return false;
    if (kind === "TITIPAN" && ["penetapan_bmmn", "usulan_peruntukan_bmmn", "persetujuan_peruntukan_bmmn"].includes(stepCode)) return false;
    if (["penelitian_pfpd", "pencacahan", "pindah_bongkar_kontainer"].includes(stepCode)) return false;
    if (stepCode === "persetujuan_peruntukan_bmmn" && !item.dataset.allocationProposal) return false;
    return true;
  }

  function filterInventoryPicker() {
    const term = (inventorySearch?.value || "").trim().toLowerCase();
    let visible = 0;
    inventoryPickerItems.forEach((item) => {
      const eligible = inventoryItemEligible(item);
      const matches = !term || (item.dataset.search || "").includes(term);
      item.hidden = !(eligible && matches);
      const checkbox = $("[data-picker-checkbox]", item);
      if (checkbox) {
        checkbox.disabled = !eligible;
        if (!eligible) checkbox.checked = false;
      }
      if (!item.hidden) visible++;
    });
    const empty = $("[data-picker-empty]", inventoryForm);
    if (empty) empty.hidden = visible > 0;
    reorderSelectionGroup(inventoryPickerList, inventoryPickerItems, (item) => $('[data-picker-checkbox]', item)?.checked);
    updateInventorySelection();
  }

  function updateExitOptions(selected) {
    const select = $("[data-exit-type]", inventoryForm);
    if (!select) return;
    const types = [...new Set(selected.map((item) => item.dataset.itemType))];
    $$("option[data-types]", select).forEach((option) => {
      const allowed = (option.dataset.types || "").split(",");
      const processCompatible = selected.every((item) => {
        const status = item.dataset.itemStatus || "";
        if (status === "laku" || status === "alokasi_hasil_lelang") return option.value === "lelang";
        if (status === "kep_musnah" || status === "ba_musnah" || item.dataset.currentDisposition === "musnah") return option.value === "musnah";
        if (status === "ba_serah_terima") {
          const label = (item.dataset.itemStatusLabel || "").replace(/^BA Serah Terima\s*/i, "").trim().toLowerCase();
          const optionLabel = (option.textContent || "").trim().toLowerCase();
          return option.value.toLowerCase() === label || optionLabel === label;
        }
        return true;
      });
      const visible = (types.length === 0 || types.every((type) => allowed.includes(type))) && processCompatible;
      option.hidden = !visible;
      option.disabled = !visible;
      if (!visible && option.selected) select.value = "";
    });
  }

  function setPFPDRestrictionState(row) {
    const select = $('[data-pfpd-field="is_restricted"]', row);
    const wrapper = $('[data-pfpd-restriction-wrapper]', row);
    const input = $('[data-pfpd-field="restriction_rule"]', row);
    const visible = select?.value === 'ya' && !select.disabled;
    if (wrapper) wrapper.hidden = !visible;
    if (input) {
      input.disabled = !visible;
      input.required = visible;
      if (!visible) input.value = '';
    }
  }

  function filterPFPDRequests() {
    const term = (pfpdRequestSearch?.value || '').trim().toLowerCase();
    let visible = 0;
    let activeWasHidden = false;
    pfpdRequestCards.forEach((card) => {
      const matches = !term || (card.dataset.pfpdSearch || '').includes(term);
      card.hidden = !matches;
      if (matches) visible++;
      if (!matches && card === activePFPDRequestCard) activeWasHidden = true;
    });
    if (activeWasHidden) activatePFPDRequest(null);
    if (pfpdSearchEmpty) pfpdSearchEmpty.hidden = visible > 0 || pfpdRequestCards.length === 0;
    reorderSelectionGroup(pfpdRequestList, pfpdRequestCards, () => false, activePFPDRequestCard);
  }

  function activatePFPDRequest(card) {
    activePFPDRequestCard = card;
    pfpdRequestCards.forEach((candidate) => {
      const active = candidate === card;
      candidate.classList.toggle('active', active);
      const items = $('[data-pfpd-request-items]', candidate);
      if (items) items.hidden = !active;
      $$('input, select, textarea', candidate).forEach((field) => { field.disabled = !active; });
      if (active) $$('[data-pfpd-result-row]', candidate).forEach(setPFPDRestrictionState);
    });
    const count = $('[data-pfpd-picker-count]', inventoryForm);
    if (count) count.textContent = card ? '1 request dipilih' : '0 request dipilih';
    const submit = $('[data-submit-step]', inventoryForm);
    if (submit) submit.disabled = !card;
    reorderSelectionGroup(pfpdRequestList, pfpdRequestCards, () => false, activePFPDRequestCard);
  }

  function serializePFPDResults() {
    if (!activePFPDRequestCard || !pfpdResultsJSON) return false;
    const rows = $$('[data-pfpd-result-row]', activePFPDRequestCard);
    const results = rows.map((row) => ({
      inventory_id: row.dataset.inventoryId || '',
      hs_code: $('[data-pfpd-field="hs_code"]', row)?.value.trim() || '',
      is_restricted: $('[data-pfpd-field="is_restricted"]', row)?.value || '',
      restriction_rule: $('[data-pfpd-field="restriction_rule"]', row)?.value.trim() || '',
      goods_value: $('[data-pfpd-field="goods_value"]', row)?.value.trim() || '',
    }));
    if (!results.length || results.some((result) => !result.inventory_id || !result.hs_code || !result.is_restricted || !result.goods_value || (result.is_restricted === 'ya' && !result.restriction_rule))) return false;
    pfpdResultsJSON.value = JSON.stringify(results);
    pfpdResultsJSON.disabled = false;
    return true;
  }

  function updateInventorySelection() {
    const selected = selectedInventoryPickerItems();
    const count = $("[data-picker-count]", inventoryForm);
    if (count) count.textContent = `${selected.length} barang dipilih`;
    if (inventorySelectAll) {
      const visible = inventoryPickerItems.filter((item) => !item.hidden && !$("[data-picker-checkbox]", item)?.disabled);
      inventorySelectAll.checked = visible.length > 0 && visible.every((item) => $("[data-picker-checkbox]", item).checked);
      inventorySelectAll.indeterminate = visible.some((item) => $("[data-picker-checkbox]", item).checked) && !inventorySelectAll.checked;
    }
    if (inventoryStep?.dataset.stepCode === "pengeluaran_barang") updateExitOptions(selected);

    const submit = $("[data-submit-step]", inventoryForm);
    if (submit && !["penelitian_pfpd", "pencacahan", "pindah_bongkar_kontainer"].includes(inventoryStep?.dataset.stepCode || "")) submit.disabled = !inventoryStep || selected.length === 0;
    reorderSelectionGroup(inventoryPickerList, inventoryPickerItems, (item) => $('[data-picker-checkbox]', item)?.checked);
  }

  function showInventoryStep(step) {
    inventoryStep = step;
    activePFPDRequestCard = null;
    $$('[data-picker-checkbox]', inventoryForm).forEach((box) => { box.checked = false; });
    censusTargetCards.forEach((card) => {
      const box = $('[data-census-target-checkbox]', card);
      if (box) box.checked = false;
      $$('[data-census-result-line][data-existing="false"]', card).forEach((row) => row.remove());
      setCensusTargetState(card);
    });
    const picker = $("[data-inventory-picker]", inventoryForm);
    const fields = $("[data-action-fields]", inventoryForm);
    const isPFPD = step.dataset.stepCode === "penelitian_pfpd";
    const isCensus = step.dataset.stepCode === "pencacahan";
    const isRelocation = step.dataset.stepCode === "pindah_bongkar_kontainer";
    if (picker) picker.hidden = isPFPD || isCensus || isRelocation;
    if (pfpdRequestPicker) pfpdRequestPicker.hidden = !isPFPD;
    if (pfpdResultsJSON) pfpdResultsJSON.disabled = !isPFPD;
    if (censusTargetPicker) censusTargetPicker.hidden = !isCensus;
    if (censusResultsJSON) censusResultsJSON.disabled = !isCensus;
    if (relocationPicker) relocationPicker.hidden = !isRelocation;
    if (relocationJSON) relocationJSON.disabled = !isRelocation;
    if (fields) fields.hidden = false;
    $$('input[name="document_no"], input[name="document_date"], input[name="document_file"]', inventoryForm).forEach((field) => { field.disabled = false; });
    $$('[data-fields-for]', inventoryForm).forEach((section) => {
      const active = section.dataset.fieldsFor === step.dataset.stepCode;
      section.hidden = !active;
      $$('input, select, textarea', section).forEach((field) => { field.disabled = !active; });
    });
    if (isPFPD) activatePFPDRequest(null);
    if (isRelocation) {
      relocationSources.forEach((source) => {
        source.classList.remove('selected');
        const checkbox = $('[data-relocation-source-checkbox]', source);
        if (checkbox) checkbox.checked = false;
      });
      if (relocationEditorList) relocationEditorList.replaceChildren();
      if (relocationMode) relocationMode.value = 'bongkar';
      if (relocationSearch) relocationSearch.value = '';
      filterRelocationSources();
    }
    if (isCensus) {
      if (censusTargetSearch) censusTargetSearch.value = "";
      filterCensusTargets();
      updateCensusTargetSelection();
    }
    if (isPFPD) {
      if (pfpdRequestSearch) pfpdRequestSearch.value = "";
      filterPFPDRequests();
    }
    if (inventorySearch) inventorySearch.value = "";
    updateLartasFields();
    filterInventoryPicker();
  }

  bindStepPicker(inventoryForm, showInventoryStep);
  $$('[data-open-inventory-action]').forEach((button) => button.addEventListener("click", () => {
    inventoryStep = null;
    activePFPDRequestCard = null;
    resetActionForm(inventoryForm);
    censusTargetCards.forEach((card) => { const box = $('[data-census-target-checkbox]', card); if (box) box.checked = false; setCensusTargetState(card); });
    inventoryPickerItems.forEach((item) => { item.hidden = false; });
    openModal(inventoryDrawer);
  }));
  $("[data-back-action-menu]", inventoryForm)?.addEventListener("click", () => {
    inventoryStep = null;
    activePFPDRequestCard = null;
    resetActionForm(inventoryForm);
    censusTargetCards.forEach((card) => { const box = $('[data-census-target-checkbox]', card); if (box) box.checked = false; setCensusTargetState(card); });
    inventoryPickerItems.forEach((item) => { item.hidden = false; });
  });
  inventorySearch?.addEventListener("input", filterInventoryPicker);
  pfpdRequestSearch?.addEventListener("input", filterPFPDRequests);
  relocationSearch?.addEventListener('input', filterRelocationSources);
  relocationMode?.addEventListener('change', () => {
    relocationSources.forEach((source) => {
      const box = $('[data-relocation-source-checkbox]', source);
      if (box && !relocationSourceMatchesMode(source)) box.checked = false;
    });
    if (relocationEditorList) relocationEditorList.replaceChildren();
    filterRelocationSources();
  });
  relocationSources.forEach((source) => {
    $('[data-relocation-source-checkbox]', source)?.addEventListener('change', updateRelocationSelection);
  });
  inventoryPickerItems.forEach((item) => $("[data-picker-checkbox]", item)?.addEventListener("change", updateInventorySelection));
  inventorySelectAll?.addEventListener("change", () => {
    inventoryPickerItems.filter((item) => !item.hidden).forEach((item) => {
      const box = $("[data-picker-checkbox]", item);
      if (box && !box.disabled) box.checked = inventorySelectAll.checked;
    });
    updateInventorySelection();
  });
  $("[data-picker-clear]", inventoryForm)?.addEventListener("click", () => {
    inventoryPickerItems.forEach((item) => {
      const box = $("[data-picker-checkbox]", item);
      if (box) box.checked = false;
    });
    updateInventorySelection();
  });

  censusTargetSearch?.addEventListener("input", filterCensusTargets);
  censusTargetCards.forEach((card) => {
    $('[data-census-target-checkbox]', card)?.addEventListener('change', updateCensusTargetSelection);
    $('[data-add-census-target-line]', card)?.addEventListener('click', () => addCensusTargetLine(card));
  });
  pfpdRequestCards.forEach((card) => {
    $("[data-pfpd-request-button]", card)?.addEventListener("click", () => activatePFPDRequest(card));
    $$('[data-pfpd-field="is_restricted"]', card).forEach((select) => select.addEventListener("change", () => setPFPDRestrictionState(select.closest("[data-pfpd-result-row]"))));
  });

  const lartasSelect = $("[data-lartas-select]", inventoryForm);
  function updateLartasFields() {
    const notes = $("[data-lartas-notes]", inventoryForm);
    const input = $('input[name="restriction_rule"]', notes);
    const visible = lartasSelect?.value === "ya" && !lartasSelect.disabled;
    if (notes) notes.hidden = !visible;
    if (input) {
      input.disabled = !visible;
      input.required = visible;
      if (!visible) input.value = "";
    }
  }
  lartasSelect?.addEventListener("change", updateLartasFields);
  inventoryForm?.addEventListener("submit", (event) => {
    const code = $("[data-event-code]", inventoryForm)?.value;
    if (code === "pencacahan" && !serializeCensusResults()) {
      event.preventDefault();
      window.alert("Pilih target dan lengkapi uraian, jenis, jumlah, satuan, serta kondisi setiap barang hasil pencacahan.");
      return;
    }

    if (code === "penelitian_pfpd" && !serializePFPDResults()) {
      event.preventDefault();
      window.alert("Buka satu nomor request dan lengkapi HS code, nilai, serta status lartas seluruh uraian.");
      return;
    }
    if (code === "pindah_bongkar_kontainer" && !serializeRelocation()) {
      event.preventDefault();
      window.alert("Pilih satu atau beberapa target bongkar/muat, lengkapi tujuan masing-masing target, lalu pastikan total kuantitas setiap target sama dengan kuantitas sumbernya.");
      return;
    }
    if (code === "pindah_bongkar_kontainer" && !window.confirm("Data bongkar/muat kontainer akan diproses sesuai tujuan yang diisi. Nomor kontainer, kuantitas, dan perhitungan YOR/SOR akan diperbarui. Lanjutkan?")) {
      event.preventDefault();
      return;
    }
    if (code === "penetapan_bmmn" && !window.confirm("Seluruh barang yang dipilih akan berubah menjadi BMMN. Lanjutkan?")) event.preventDefault();
    if (code === "pengeluaran_barang" && !window.confirm("Seluruh barang yang dipilih akan dikeluarkan dari inventory aktif. Lanjutkan?")) event.preventDefault();
  });

  const processDrawer = $("#process-action-drawer");
  const processForm = $("[data-process-bulk-form]", processDrawer);
  const processCandidates = $$('[data-process-candidate]', processForm);
  const processPickerList = $('[data-process-picker-list]', processForm);
  const processSearch = $("[data-process-picker-search]", processForm);
  const processSelectAll = $("[data-process-picker-select-all]", processForm);
  const auctionSchedulePicker = $("[data-auction-schedule-picker]", processForm);
  const auctionScheduleCards = $$('[data-auction-schedule-card]', processForm);
  const auctionScheduleList = $('.auction-schedule-list', processForm);
  const auctionScheduleSearch = $('[data-auction-schedule-search]', processForm);
  const auctionScheduleClear = $('[data-auction-schedule-clear]', processForm);
  const auctionScheduleVisible = $('[data-auction-schedule-visible]', processForm);
  const auctionScheduleNo = $('[data-auction-schedule-no]', processForm);
  const auctionResultsJSON = $('[data-auction-results-json]', processForm);
  const htlResultsJSON = $('[data-htl-results-json]', processForm);
  const htlEditorList = $('[data-htl-editor-list]', processForm);
  let processStep = null;
  let activeAuctionSchedule = null;

  function selectedProcessCandidates() {
    return processCandidates.filter((item) => $("[data-process-candidate-checkbox]", item)?.checked);
  }

  function processCandidateEligible(item) {
    if (!processStep) return false;
    const source = processStep.dataset.createsProcess === "true" ? "inventory" : "process";
    if (item.dataset.candidateSource !== source) return false;
    if (source === "inventory") return true;
    if (item.dataset.active !== "true") return false;
    const allowed = (processStep.dataset.allowedStatus || "").split(",").filter(Boolean);
    return allowed.includes(item.dataset.status || "");
  }

  function filterProcessCandidates() {
    const term = (processSearch?.value || "").trim().toLowerCase();
    let visible = 0;
    processCandidates.forEach((item) => {
      const eligible = processCandidateEligible(item);
      const matches = !term || (item.dataset.search || "").includes(term);
      item.hidden = !(eligible && matches);
      const checkbox = $("[data-process-candidate-checkbox]", item);
      if (checkbox) {
        checkbox.disabled = !eligible;
        if (!eligible) checkbox.checked = false;
      }
      if (!item.hidden) visible++;
    });
    const empty = $("[data-process-picker-empty]", processForm);
    if (empty) empty.hidden = visible > 0;
    reorderSelectionGroup(processPickerList, processCandidates, (item) => $('[data-process-candidate-checkbox]', item)?.checked);
    updateProcessSelection();
  }

  function renderHTLEditors() {
    if (!htlEditorList || processStep?.dataset.stepCode !== 'kep_htl') return;
    const saved = new Map($$('[data-htl-editor-row]', htlEditorList).map((row) => [row.dataset.processId, $('[data-htl-value]', row)?.value || '']));
    htlEditorList.replaceChildren();
    const selected = selectedProcessCandidates();
    if (!selected.length) {
      const help = document.createElement('p');
      help.className = 'field-help';
      help.textContent = 'Pilih satu atau beberapa barang. Setiap barang akan memperoleh input nilai HTL tersendiri.';
      htlEditorList.append(help);
      return;
    }
    selected.forEach((candidate, index) => {
      const processID = candidate.dataset.processId || $('[data-process-candidate-checkbox]', candidate)?.value || '';
      const row = document.createElement('section');
      row.className = 'per-item-value-row';
      row.dataset.htlEditorRow = '';
      row.dataset.processId = processID;
      const header = document.createElement('header');
      const copy = document.createElement('div');
      const strong = document.createElement('strong');
      strong.textContent = candidate.dataset.containerLabel || `Barang ${index + 1}`;
      const small = document.createElement('small');
      small.textContent = `${candidate.dataset.determination || ''} · ${candidate.dataset.description || ''}`;
      copy.append(strong, small);
      header.append(copy);
      const label = document.createElement('label');
      label.innerHTML = 'Nilai HTL <em>*</em>';
      const inputShell = document.createElement('span');
      inputShell.className = 'currency-input-shell';
      const prefix = document.createElement('span');
      prefix.textContent = 'Rp';
      const input = document.createElement('input');
      input.inputMode = 'numeric';
      input.placeholder = 'Nilai HTL dalam rupiah';
      input.dataset.htlValue = '';
      input.required = true;
      input.value = saved.get(processID) || '';
      input.addEventListener('input', updateHTLValidity);
      inputShell.append(prefix, input);
      label.append(inputShell);
      row.append(header, label);
      htlEditorList.append(row);
    });
  }

  function serializeHTLResults() {
    if (!htlResultsJSON) return false;
    const results = $$('[data-htl-editor-row]', htlEditorList).map((row) => ({
      process_id: row.dataset.processId || '',
      htl_value: $('[data-htl-value]', row)?.value.trim() || '',
    }));
    const valid = results.length > 0 && results.every((result) => result.process_id && Number(String(result.htl_value).replace(/[^0-9]/g, '')) > 0);
    if (valid) {
      htlResultsJSON.value = JSON.stringify(results);
      htlResultsJSON.disabled = false;
    }
    return valid;
  }

  function updateHTLValidity() {
    const submit = $('[data-submit-step]', processForm);
    if (!submit || processStep?.dataset.stepCode !== 'kep_htl') return;
    const rows = $$('[data-htl-editor-row]', htlEditorList);
    submit.disabled = !rows.length || rows.some((row) => Number(String($('[data-htl-value]', row)?.value || '').replace(/[^0-9]/g, '')) <= 0);
  }

  function updateProcessSelection() {
    if (processStep?.dataset.stepCode === "selesai_lelang" && auctionSchedulePicker) {
      updateAuctionScheduleValidity();
      return;
    }
    const selected = selectedProcessCandidates();
    const count = $("[data-process-picker-count]", processForm);
    if (count) count.textContent = `${selected.length} barang dipilih`;
    if (processStep?.dataset.stepCode === "kep_htl") renderHTLEditors();
    if (processSelectAll) {
      const visible = processCandidates.filter((item) => !item.hidden && !$("[data-process-candidate-checkbox]", item)?.disabled);
      processSelectAll.checked = visible.length > 0 && visible.every((item) => $("[data-process-candidate-checkbox]", item).checked);
      processSelectAll.indeterminate = visible.some((item) => $("[data-process-candidate-checkbox]", item).checked) && !processSelectAll.checked;
    }
    const submit = $("[data-submit-step]", processForm);
    if (submit) {
      if (processStep?.dataset.stepCode === "kep_htl") updateHTLValidity();
      else submit.disabled = !processStep || selected.length === 0;
    }
    reorderSelectionGroup(processPickerList, processCandidates, (item) => $('[data-process-candidate-checkbox]', item)?.checked);
  }

  function resetAuctionSchedules() {
    activeAuctionSchedule = null;
    if (auctionScheduleNo) {
      auctionScheduleNo.value = "";
      auctionScheduleNo.disabled = true;
    }
    if (auctionResultsJSON) {
      auctionResultsJSON.value = "";
      auctionResultsJSON.disabled = true;
    }
    if (auctionScheduleSearch) auctionScheduleSearch.value = "";
    if (auctionScheduleClear) auctionScheduleClear.disabled = true;
    auctionScheduleCards.forEach((card) => {
      card.hidden = false;
      card.classList.remove("active");
      const list = $('[data-auction-result-list]', card);
      if (list) list.hidden = true;
      $$('select, input', card).forEach((field) => {
        field.disabled = true;
        field.required = false;
        field.value = "";
      });
      $$('[data-auction-result-sale]', card).forEach((wrapper) => { wrapper.hidden = true; });
    });
    const count = $('[data-auction-schedule-count]', processForm);
    if (count) count.textContent = '0 ND dipilih';
    const empty = $('[data-auction-schedule-empty]', processForm);
    if (empty) empty.hidden = true;
  }

  function setAuctionResultState(row) {
    const outcome = $('[data-auction-result-outcome]', row);
    const wrapper = $('[data-auction-result-sale]', row);
    const sale = $('[data-auction-result-sale-input]', row);
    const active = row.closest('[data-auction-schedule-card]') === activeAuctionSchedule;
    const isSold = active && outcome?.value === 'laku';
    if (wrapper) wrapper.hidden = !isSold;
    if (sale) {
      sale.disabled = !isSold;
      sale.required = isSold;
      if (!isSold) sale.value = '';
    }
    updateAuctionScheduleValidity();
  }

  function activateAuctionSchedule(card) {
    activeAuctionSchedule = card;
    auctionScheduleCards.forEach((candidate) => {
      const active = candidate === card;
      candidate.classList.toggle('active', active);
      const list = $('[data-auction-result-list]', candidate);
      if (list) list.hidden = !active;
      $$('[data-auction-result-outcome]', candidate).forEach((field) => {
        field.disabled = !active;
        field.required = active;
        if (!active) field.value = '';
      });
      $$('[data-auction-result-row]', candidate).forEach(setAuctionResultState);
    });
    if (auctionScheduleNo) {
      auctionScheduleNo.value = card?.dataset.scheduleNo || '';
      auctionScheduleNo.disabled = !card;
    }
    if (auctionResultsJSON) auctionResultsJSON.disabled = !card;
    const count = $('[data-auction-schedule-count]', processForm);
    if (count) count.textContent = card ? '1 ND dipilih' : '0 ND dipilih';
    reorderSelectionGroup(auctionScheduleList, auctionScheduleCards, () => false, activeAuctionSchedule);
    updateAuctionScheduleValidity();
  }

  function filterAuctionSchedules() {
    const term = (auctionScheduleSearch?.value || '').trim().toLowerCase();
    let visible = 0;
    let activeHidden = false;
    auctionScheduleCards.forEach((card) => {
      const matches = !term || (card.dataset.auctionScheduleSearchValue || '').includes(term);
      card.hidden = !matches;
      if (matches) visible++;
      if (!matches && card === activeAuctionSchedule) activeHidden = true;
    });
    if (activeHidden) activateAuctionSchedule(null);
    if (auctionScheduleVisible) auctionScheduleVisible.textContent = `${visible} ND tersedia`;
    if (auctionScheduleClear) auctionScheduleClear.disabled = !term;
    const empty = $('[data-auction-schedule-empty]', processForm);
    if (empty) empty.hidden = visible > 0 || auctionScheduleCards.length === 0;
    reorderSelectionGroup(auctionScheduleList, auctionScheduleCards, () => false, activeAuctionSchedule);
  }

  function serializeAuctionResults() {
    if (!activeAuctionSchedule || !auctionResultsJSON) return false;
    const results = $$('[data-auction-result-row]', activeAuctionSchedule).map((row) => ({
      process_id: row.dataset.processId || '',
      outcome: $('[data-auction-result-outcome]', row)?.value || '',
      sale_value: $('[data-auction-result-sale-input]', row)?.value.trim() || '',
    }));
    const valid = results.length > 0 && results.every((result) => result.process_id && (result.outcome === 'laku' || result.outcome === 'tidak_laku') && (result.outcome !== 'laku' || Number(String(result.sale_value).replace(/[^0-9]/g, '')) > 0));
    if (valid) auctionResultsJSON.value = JSON.stringify(results);
    return valid;
  }

  function updateAuctionScheduleValidity() {
    const submit = $('[data-submit-step]', processForm);
    if (!submit || processStep?.dataset.stepCode !== 'selesai_lelang') return;
    const rows = activeAuctionSchedule ? $$('[data-auction-result-row]', activeAuctionSchedule) : [];
    const complete = rows.length > 0 && rows.every((row) => {
      const outcome = $('[data-auction-result-outcome]', row)?.value || '';
      const sale = $('[data-auction-result-sale-input]', row)?.value || '';
      return (outcome === 'laku' && Number(String(sale).replace(/[^0-9]/g, '')) > 0) || outcome === 'tidak_laku';
    });
    submit.disabled = !complete;
  }

  function showProcessStep(step) {
    processStep = step;
    $$('[data-process-candidate-checkbox]', processForm).forEach((box) => { box.checked = false; });
    resetAuctionSchedules();
    if (htlResultsJSON) { htlResultsJSON.value = ""; htlResultsJSON.disabled = true; }
    if (htlEditorList) htlEditorList.replaceChildren();
    const picker = $("[data-process-picker]", processForm);
    const fields = $("[data-process-action-fields]", processForm);
    const groupedAuctionResult = step.dataset.stepCode === 'selesai_lelang' && !!auctionSchedulePicker;
    if (picker) picker.hidden = groupedAuctionResult;
    if (auctionSchedulePicker) auctionSchedulePicker.hidden = !groupedAuctionResult;
    if (fields) fields.hidden = false;
    $$('input[name="document_no"], input[name="document_date"], input[name="document_file"]', processForm).forEach((field) => { field.disabled = false; });
    $$('[data-process-fields-for]', processForm).forEach((section) => {
      const active = section.dataset.processFieldsFor === step.dataset.stepCode;
      section.hidden = !active;
      $$('input, select, textarea', section).forEach((field) => { field.disabled = !active; });
    });
    if (groupedAuctionResult) {
      if (auctionScheduleNo) auctionScheduleNo.disabled = true;
      if (auctionResultsJSON) auctionResultsJSON.disabled = true;
      filterAuctionSchedules();
      updateAuctionScheduleValidity();
    } else {
      if (processSearch) processSearch.value = "";
      filterProcessCandidates();
      if (step.dataset.stepCode === "kep_htl") {
        if (htlResultsJSON) htlResultsJSON.disabled = false;
        renderHTLEditors();
        updateHTLValidity();
      }
    }
  }

  bindStepPicker(processForm, showProcessStep);
  $$('[data-open-process-action]').forEach((button) => button.addEventListener("click", () => {
    processStep = null;
    resetActionForm(processForm);
    resetAuctionSchedules();
    openModal(processDrawer);
  }));
  $("[data-back-action-menu]", processForm)?.addEventListener("click", () => {
    processStep = null;
    resetActionForm(processForm);
    resetAuctionSchedules();
  });
  processSearch?.addEventListener("input", filterProcessCandidates);
  processCandidates.forEach((item) => $("[data-process-candidate-checkbox]", item)?.addEventListener("change", updateProcessSelection));
  processSelectAll?.addEventListener("change", () => {
    processCandidates.filter((item) => !item.hidden).forEach((item) => {
      const box = $("[data-process-candidate-checkbox]", item);
      if (box && !box.disabled) box.checked = processSelectAll.checked;
    });
    updateProcessSelection();
  });
  $("[data-process-picker-clear]", processForm)?.addEventListener("click", () => {
    processCandidates.forEach((item) => {
      const box = $("[data-process-candidate-checkbox]", item);
      if (box) box.checked = false;
    });
    updateProcessSelection();
  });
  auctionScheduleSearch?.addEventListener('input', filterAuctionSchedules);
  auctionScheduleClear?.addEventListener('click', () => {
    if (auctionScheduleSearch) {
      auctionScheduleSearch.value = '';
      auctionScheduleSearch.focus();
    }
    filterAuctionSchedules();
  });
  auctionScheduleCards.forEach((card) => {
    $('[data-auction-schedule-button]', card)?.addEventListener('click', () => activateAuctionSchedule(card));
    $$('[data-auction-result-outcome]', card).forEach((select) => select.addEventListener('change', () => setAuctionResultState(select.closest('[data-auction-result-row]'))));
    $$('[data-auction-result-sale-input]', card).forEach((input) => input.addEventListener('input', updateAuctionScheduleValidity));
  });

  processForm?.addEventListener("submit", (event) => {
    const groupedAuctionResult = processStep?.dataset.stepCode === 'selesai_lelang' && !!auctionSchedulePicker;
    let count = selectedProcessCandidates().length;
    if (groupedAuctionResult) {
      count = activeAuctionSchedule ? $$('[data-auction-result-row]', activeAuctionSchedule).length : 0;
      if (!serializeAuctionResults()) {
        event.preventDefault();
        window.alert('Pilih satu ND penjadwalan dan lengkapi hasil setiap barang. Harga jual wajib diisi untuk barang yang laku.');
        return;
      }
    }
    if (processStep?.dataset.stepCode === 'kep_htl' && !serializeHTLResults()) {
      event.preventDefault();
      window.alert('Isi nilai HTL masing-masing barang yang dipilih.');
      return;
    }
    const label = processStep?.dataset.stepLabel || "Action";
    if (!window.confirm(`${label} akan diterapkan pada ${count} barang. Lanjutkan?`)) event.preventDefault();
  });

  const reconciliationModal = $('#reconciliation-modal');
  const reconciliationForm = $('[data-reconciliation-form]', reconciliationModal);
  const reconciliationSections = $$('[data-reconciliation-fields]', reconciliationForm);
  const reconciliationItems = $$('[data-reconciliation-item]', reconciliationForm);
  const reconciliationSearch = $('[data-reconciliation-search]', reconciliationForm);
  const reconciliationSearchClear = $('[data-reconciliation-search-clear]', reconciliationForm);
  const reconciliationPickerCount = $('[data-reconciliation-picker-count]', reconciliationForm);
  const reconciliationEmpty = $('[data-reconciliation-empty]', reconciliationForm);
  const reconciliationPickerTitle = $('[data-reconciliation-picker-title]', reconciliationForm);
  const reconciliationPickerHelp = $('[data-reconciliation-picker-help]', reconciliationForm);
  const reconciliationNotes = $('[data-reconciliation-notes]', reconciliationForm);
  const reconciliationSubmit = $('[data-reconciliation-submit]', reconciliationForm);
  const correctionLoading = $('[data-correction-loading]', reconciliationForm);
  const correctionEmpty = $('[data-correction-empty]', reconciliationForm);
  const correctionEditor = $('[data-correction-editor]', reconciliationForm);
  const correctionEventList = $('[data-correction-event-list]', reconciliationForm);
  const correctionProcessList = $('[data-correction-process-list]', reconciliationForm);
  const correctionItemJSON = $('[data-correction-item-json]', reconciliationForm);
  const correctionEventsJSON = $('[data-correction-events-json]', reconciliationForm);
  const correctionProcessesJSON = $('[data-correction-processes-json]', reconciliationForm);
  const correctionReason = $('[data-correction-reason]', reconciliationForm);
  const correctionUpload = $('[data-correction-upload]', reconciliationForm);
  const reconciliationTypeSection = $('[data-reconciliation-type-section]', reconciliationForm);
  const reconciliationCorrectionOption = $('[data-reconciliation-correction-option]', reconciliationForm);
  const reconciliationModalEyebrow = $('[data-reconciliation-modal-eyebrow]', reconciliationModal);
  const reconciliationModalTitle = $('[data-reconciliation-modal-title]', reconciliationModal);
  const reconciliationModalDescription = $('[data-reconciliation-modal-description]', reconciliationModal);
  let correctionState = null;
  let correctionLoadToken = 0;

  function reconciliationType() {
    return $('input[name="reconciliation_type"]:checked', reconciliationForm)?.value || '';
  }

  function sectionSupportsType(section, type) {
    return (section.dataset.reconciliationFields || '').split(',').map((value) => value.trim()).includes(type);
  }

  function setCorrectionEnabled(enabled) {
    $$('[data-correction-field], [data-correction-event-field], [data-correction-process-field]', correctionEditor).forEach((field) => {
      field.disabled = !enabled;
      field.required = enabled && field.dataset.required === 'true';
    });
    if (correctionReason) {
      correctionReason.disabled = !enabled;
      correctionReason.required = enabled;
    }
    if (correctionUpload) correctionUpload.disabled = !enabled;
    [correctionItemJSON, correctionEventsJSON, correctionProcessesJSON].forEach((field) => {
      if (field) field.disabled = !enabled;
    });
  }

  function resetCorrectionEditor() {
    correctionState = null;
    correctionLoadToken++;
    if (correctionLoading) correctionLoading.hidden = true;
    if (correctionEmpty) correctionEmpty.hidden = false;
    if (correctionEditor) correctionEditor.hidden = true;
    if (correctionEventList) correctionEventList.replaceChildren();
    if (correctionProcessList) correctionProcessList.replaceChildren();
    [correctionItemJSON, correctionEventsJSON, correctionProcessesJSON].forEach((field) => {
      if (field) field.value = '';
    });
    setCorrectionEnabled(false);
  }

  function updateReconciliationType(type) {
    const mandatoryFoundFields = new Set(['determination_no', 'determination_date', 'item_type', 'facility_id', 'load_type', 'description', 'item_kind', 'quantity', 'unit', 'initial_status_code']);
    reconciliationSections.forEach((section) => {
      const active = sectionSupportsType(section, type);
      section.hidden = !active;
      $$('input, select, textarea', section).forEach((field) => {
        const isCorrectionControl = field.matches('[data-correction-field], [data-correction-event-field], [data-correction-process-field], [data-correction-reason], [data-correction-upload], [data-correction-item-json], [data-correction-events-json], [data-correction-processes-json]');
        if (isCorrectionControl) return;
        field.disabled = !active;
        field.required = active && ((type === 'recorded_not_found' || type === 'data_correction') && field.name === 'inventory_id' || type === 'found_not_recorded' && mandatoryFoundFields.has(field.name));
      });
    });
    const correctionMode = type === 'data_correction';
    if (reconciliationNotes) {
      reconciliationNotes.hidden = correctionMode;
      $$('input, select, textarea', reconciliationNotes).forEach((field) => {
        field.disabled = correctionMode;
        field.required = !correctionMode && field.name === 'notes';
      });
    }
    if (reconciliationPickerTitle) reconciliationPickerTitle.textContent = correctionMode ? 'Pilih barang yang datanya akan diubah' : 'Pilih inventory yang tidak ditemukan';
    if (reconciliationPickerHelp) reconciliationPickerHelp.textContent = correctionMode ? 'Setelah barang dipilih, seluruh data bisnis, nomor dokumen, dan data proses akan dimuat untuk diperbarui.' : 'Cari berdasarkan nomor penetapan, nomor kontainer, jenis inventory, atau uraian barang.';
    if (!correctionMode) resetCorrectionEditor();
    updateReconciliationItemType();
    updateReconciliationLoadType();
  }

  function updateReconciliationItemType() {
    const type = $('[data-reconciliation-item-type]', reconciliationForm)?.value || '';
    const isBDN = type === 'BDN';
    const isTitipan = type === 'TITIPAN';
    $$('[data-reconciliation-bdn]', reconciliationForm).forEach((wrapper) => {
      wrapper.hidden = !isBDN;
      $$('input, select', wrapper).forEach((field) => { field.disabled = !isBDN; field.required = isBDN; if (!isBDN) field.value = ''; });
    });
    $$('[data-reconciliation-titipan]', reconciliationForm).forEach((wrapper) => {
      wrapper.hidden = !isTitipan;
      $$('input, select', wrapper).forEach((field) => { field.disabled = !isTitipan; field.required = isTitipan; if (!isTitipan) field.value = ''; });
    });
    const tps = $('[data-reconciliation-tps]', reconciliationForm);
    if (tps) {
      tps.hidden = isTitipan || type === 'BMMN';
      $$('select, input', tps).forEach((field) => { field.disabled = tps.hidden; field.required = !tps.hidden && !!type; if (tps.hidden) field.value = ''; });
    }
  }

  function updateReconciliationLoadType() {
    const load = $('[data-reconciliation-load]', reconciliationForm)?.value || '';
    $$('[data-reconciliation-fcl]', reconciliationForm).forEach((wrapper) => {
      const active = load === 'FCL';
      wrapper.hidden = !active;
      $$('input, select', wrapper).forEach((field) => { field.disabled = !active; field.required = active; if (!active) field.value = ''; });
    });
    $$('[data-reconciliation-lcl]', reconciliationForm).forEach((wrapper) => {
      const active = load === 'LCL';
      wrapper.hidden = !active;
      $$('input, select', wrapper).forEach((field) => { field.disabled = !active; field.required = active; if (!active) field.value = ''; });
    });
  }

  function updateReconciliationPicker() {
    const term = (reconciliationSearch?.value || '').trim().toLowerCase();
    let visible = 0;
    let selected = 0;
    reconciliationItems.forEach((item) => {
      const activeAllowed = reconciliationType() !== 'recorded_not_found' || item.dataset.active === 'true';
      const matches = activeAllowed && (!term || (item.dataset.search || '').includes(term));
      item.hidden = !matches;
      const radio = $('input[name="inventory_id"]', item);
      const checked = !!radio?.checked;
      item.classList.toggle('selected', checked);
      if (checked) selected++;
      if (matches) visible++;
    });
    if (reconciliationPickerCount) reconciliationPickerCount.textContent = `${selected} barang dipilih`;
    if (reconciliationEmpty) reconciliationEmpty.hidden = visible > 0;
    if (reconciliationSearchClear) reconciliationSearchClear.disabled = !term;
  }

  function dateInputValue(value) {
    if (!value) return '';
    return String(value).slice(0, 10);
  }

  function dateJSONValue(value) {
    return value ? `${value}T00:00:00Z` : null;
  }

  function numericValue(value) {
    const number = Number.parseFloat(String(value || '').replace(',', '.'));
    return Number.isFinite(number) ? number : 0;
  }

  function moneyValue(value) {
    const digits = String(value || '').replace(/[^0-9]/g, '');
    return digits ? Number(digits) : 0;
  }

  function populateCorrectionItem(item) {
    $$('[data-correction-field]', correctionEditor).forEach((field) => {
      const key = field.dataset.correctionField;
      const value = key === 'location' ? inventoryBlockLocation(item) : item[key];
      if (field.dataset.correctionCheckbox !== undefined) field.checked = !!value;
      else if (field.dataset.correctionDate !== undefined) field.value = dateInputValue(value);
      else field.value = value ?? '';
    });
  }

  function makeCorrectionField(labelText, value, options = {}) {
    const label = document.createElement('label');
    label.textContent = labelText;
    let field;
    if (options.kind === 'textarea') {
      field = document.createElement('textarea');
      field.rows = options.rows || 2;
    } else if (options.options) {
      field = document.createElement('select');
      options.options.forEach(([optionValue, optionLabel]) => {
        const option = document.createElement('option');
        option.value = optionValue;
        option.textContent = optionLabel;
        field.append(option);
      });
    } else {
      field = document.createElement('input');
      field.type = options.kind === 'date' ? 'date' : options.kind === 'number' ? 'number' : 'text';
      if (options.kind === 'number') {
        field.min = '0';
        field.step = options.step || '1';
      }
      if (options.money) field.inputMode = 'numeric';
    }
    field.dataset[options.datasetName] = options.fieldName;
    if (options.kind === 'date') field.value = dateInputValue(value);
    else field.value = value ?? '';
    label.append(field);
    return label;
  }

  function renderCorrectionEvents(events) {
    correctionEventList?.replaceChildren();
    if (!events.length) {
      const empty = document.createElement('p');
      empty.className = 'field-help';
      empty.textContent = 'Belum ada timeline untuk barang ini.';
      correctionEventList?.append(empty);
      return;
    }
    events.forEach((event) => {
      const card = document.createElement('article');
      card.className = 'correction-record-card';
      card.dataset.correctionEvent = event.id;
      const header = document.createElement('header');
      const title = document.createElement('div');
      const strong = document.createElement('strong');
      strong.textContent = event.label || event.code || 'Tahapan';
      const small = document.createElement('small');
      small.textContent = `${event.code || 'event'} · ${dateInputValue(event.created_at) || 'tanpa tanggal sistem'}`;
      title.append(strong, small);
      header.append(title);
      const grid = document.createElement('div');
      grid.className = 'form-grid cols-2';
      grid.append(
        makeCorrectionField('Nama/label tahapan', event.label, { datasetName: 'correctionEventField', fieldName: 'label' }),
        makeCorrectionField('Nomor surat, ND, KEP, BA, atau risalah', event.document_no, { datasetName: 'correctionEventField', fieldName: 'document_no' }),
        makeCorrectionField('Tanggal dokumen', event.document_date, { datasetName: 'correctionEventField', fieldName: 'document_date', kind: 'date' }),
        makeCorrectionField('Catatan tahapan', event.notes, { datasetName: 'correctionEventField', fieldName: 'notes', kind: 'textarea' })
      );
      card.append(header, grid);
      correctionEventList?.append(card);
    });
  }

  function renderCorrectionProcesses(processes) {
    correctionProcessList?.replaceChildren();
    if (!processes.length) {
      const empty = document.createElement('p');
      empty.className = 'field-help';
      empty.textContent = 'Barang ini belum memiliki proses lelang, pemusnahan, atau hibah/PSP.';
      correctionProcessList?.append(empty);
      return;
    }
    processes.forEach((process) => {
      const card = document.createElement('article');
      card.className = 'correction-record-card';
      card.dataset.correctionProcess = process.id;
      const header = document.createElement('header');
      const title = document.createElement('div');
      const strong = document.createElement('strong');
      strong.textContent = `${String(process.disposition_type || '').toUpperCase()} · ${process.status_label || ''}`;
      const small = document.createElement('small');
      small.textContent = `Putaran ${process.round || 1} · ${process.is_active ? 'Proses aktif' : 'Riwayat proses'}`;
      title.append(strong, small);
      header.append(title);
      const grid = document.createElement('div');
      grid.className = 'form-grid cols-3';
      grid.append(
        makeCorrectionField('Jenis usulan', process.proposal_type, { datasetName: 'correctionProcessField', fieldName: 'proposal_type' }),
        makeCorrectionField('Kode penerima', process.recipient_code, { datasetName: 'correctionProcessField', fieldName: 'recipient_code' }),
        makeCorrectionField('Nama penerima', process.recipient_name, { datasetName: 'correctionProcessField', fieldName: 'recipient_name' }),
        makeCorrectionField('Nilai HTL', process.htl_value, { datasetName: 'correctionProcessField', fieldName: 'htl_value', money: true }),
        makeCorrectionField('Nilai terjual', process.sale_value, { datasetName: 'correctionProcessField', fieldName: 'sale_value', money: true }),
        makeCorrectionField('Biaya musnah', process.destruction_cost, { datasetName: 'correctionProcessField', fieldName: 'destruction_cost', money: true }),
        makeCorrectionField('Nomor ND jadwal', process.schedule_document_no, { datasetName: 'correctionProcessField', fieldName: 'schedule_document_no' }),
        makeCorrectionField('Tanggal ND jadwal', process.schedule_document_date, { datasetName: 'correctionProcessField', fieldName: 'schedule_document_date', kind: 'date' }),
        makeCorrectionField('Tanggal mulai pelaksanaan', process.execution_start_date, { datasetName: 'correctionProcessField', fieldName: 'execution_start_date', kind: 'date' }),
        makeCorrectionField('Tanggal selesai pelaksanaan', process.execution_end_date, { datasetName: 'correctionProcessField', fieldName: 'execution_end_date', kind: 'date' }),
        makeCorrectionField('Hasil lelang', process.auction_outcome, { datasetName: 'correctionProcessField', fieldName: 'auction_outcome', options: [['', 'Belum ada'], ['laku', 'Laku'], ['tidak_laku', 'Tidak laku']] }),
        makeCorrectionField('Jenis serah terima', process.transfer_type, { datasetName: 'correctionProcessField', fieldName: 'transfer_type', options: [['', 'Belum ada'], ['hibah', 'Hibah'], ['psp', 'PSP']] }),
        makeCorrectionField('Tujuan alokasi', process.allocation_target, { datasetName: 'correctionProcessField', fieldName: 'allocation_target', kind: 'textarea' })
      );
      card.append(header, grid);
      correctionProcessList?.append(card);
    });
  }

  async function loadCorrectionData(inventoryID) {
    if (!inventoryID || reconciliationType() !== 'data_correction') return;
    const token = ++correctionLoadToken;
    correctionState = null;
    if (correctionLoading) correctionLoading.hidden = false;
    if (correctionEmpty) correctionEmpty.hidden = true;
    if (correctionEditor) correctionEditor.hidden = true;
    setCorrectionEnabled(false);
    try {
      const response = await fetch(`/api/inventory/${encodeURIComponent(inventoryID)}/timeline`, { headers: { Accept: 'application/json' } });
      if (!response.ok) throw new Error('Data barang tidak dapat dimuat.');
      const data = await response.json();
      if (token !== correctionLoadToken || reconciliationType() !== 'data_correction') return;
      correctionState = { item: data.item, events: data.events || [], processes: data.processes || [] };
      populateCorrectionItem(correctionState.item);
      renderCorrectionEvents(correctionState.events);
      renderCorrectionProcesses(correctionState.processes);
      if (correctionLoading) correctionLoading.hidden = true;
      if (correctionEditor) correctionEditor.hidden = false;
      setCorrectionEnabled(true);
    } catch (error) {
      if (token !== correctionLoadToken) return;
      if (correctionLoading) correctionLoading.hidden = true;
      if (correctionEmpty) {
        correctionEmpty.hidden = false;
        const paragraph = $('p', correctionEmpty);
        if (paragraph) paragraph.textContent = error.message || 'Data barang tidak dapat dimuat.';
      }
    }
  }

  function serializeCorrection() {
    if (!correctionState || !correctionItemJSON || !correctionEventsJSON || !correctionProcessesJSON) return false;
    const item = { ...correctionState.item };
    $$('[data-correction-field]', correctionEditor).forEach((field) => {
      const key = field.dataset.correctionField;
      if (field.dataset.correctionCheckbox !== undefined) item[key] = field.checked;
      else if (field.dataset.correctionDate !== undefined) item[key] = dateJSONValue(field.value);
      else if (field.dataset.correctionMoney !== undefined) item[key] = moneyValue(field.value);
      else if (field.dataset.correctionNumber !== undefined) item[key] = numericValue(field.value);
      else item[key] = field.value.trim();
    });
    const events = $$('[data-correction-event]', correctionEventList).map((card) => {
      const result = { id: card.dataset.correctionEvent };
      $$('[data-correction-event-field]', card).forEach((field) => {
        const key = field.dataset.correctionEventField;
        result[key] = field.type === 'date' ? dateJSONValue(field.value) : field.value.trim();
      });
      return result;
    });
    const processes = $$('[data-correction-process]', correctionProcessList).map((card) => {
      const result = { id: card.dataset.correctionProcess };
      $$('[data-correction-process-field]', card).forEach((field) => {
        const key = field.dataset.correctionProcessField;
        if (field.type === 'date') result[key] = dateJSONValue(field.value);
        else if (['htl_value', 'sale_value', 'destruction_cost'].includes(key)) result[key] = moneyValue(field.value);
        else result[key] = field.value.trim();
      });
      return result;
    });
    correctionItemJSON.value = JSON.stringify(item);
    correctionEventsJSON.value = JSON.stringify(events);
    correctionProcessesJSON.value = JSON.stringify(processes);
    correctionItemJSON.disabled = false;
    correctionEventsJSON.disabled = false;
    correctionProcessesJSON.disabled = false;
    return !!item.id && !!correctionReason?.value;
  }

  $$('[data-open-reconciliation]').forEach((button) => button.addEventListener('click', () => {
    reconciliationForm?.reset();
    const correctionMode = button.dataset.reconciliationMode === 'data_correction';
    if (reconciliationTypeSection) reconciliationTypeSection.hidden = correctionMode;
    if (reconciliationCorrectionOption) reconciliationCorrectionOption.hidden = true;
    if (reconciliationModalEyebrow) reconciliationModalEyebrow.textContent = correctionMode ? 'Audit perubahan data barang' : 'Rekonsiliasi inventory';
    if (reconciliationModalTitle) reconciliationModalTitle.textContent = correctionMode ? 'Perbarui data barang dengan jejak perubahan lengkap' : 'Sesuaikan catatan dengan kondisi sebenarnya';
    if (reconciliationModalDescription) reconciliationModalDescription.textContent = correctionMode ? 'Pilih barang, ubah data yang diperlukan, lalu tentukan alasan perubahan. Sistem mencatat nilai sebelum dan sesudah.' : 'Pilih jenis rekonsiliasi, cari barang yang terkait, lalu lengkapi data pemeriksaan.';
    let selectedType = '';
    if (correctionMode) {
      const correctionRadio = $('input[name="reconciliation_type"][value="data_correction"]', reconciliationForm);
      if (correctionRadio) correctionRadio.checked = true;
      selectedType = 'data_correction';
    }
    updateReconciliationType(selectedType);
    if (reconciliationSearch) reconciliationSearch.value = '';
    reconciliationItems.forEach((item) => { item.hidden = false; item.classList.remove('selected'); });
    resetCorrectionEditor();
    updateReconciliationPicker();
    openModal(reconciliationModal);
  }));
  $$('input[name="reconciliation_type"]', reconciliationForm).forEach((radio) => radio.addEventListener('change', () => {
    updateReconciliationType(radio.value);
    updateReconciliationPicker();
    const selected = $('input[name="inventory_id"]:checked', reconciliationForm);
    if (radio.value === 'data_correction' && selected) loadCorrectionData(selected.value);
  }));
  $('[data-reconciliation-item-type]', reconciliationForm)?.addEventListener('change', updateReconciliationItemType);
  $('[data-reconciliation-load]', reconciliationForm)?.addEventListener('change', updateReconciliationLoadType);
  reconciliationSearch?.addEventListener('input', updateReconciliationPicker);
  reconciliationSearchClear?.addEventListener('click', () => {
    if (reconciliationSearch) {
      reconciliationSearch.value = '';
      reconciliationSearch.focus();
    }
    updateReconciliationPicker();
  });
  reconciliationItems.forEach((item) => $('input[name="inventory_id"]', item)?.addEventListener('change', (event) => {
    updateReconciliationPicker();
    if (reconciliationType() === 'data_correction') loadCorrectionData(event.currentTarget.value);
  }));
  reconciliationForm?.addEventListener('submit', (event) => {
    if (reconciliationType() !== 'data_correction') return;
    if (!serializeCorrection()) {
      event.preventDefault();
      window.alert('Pilih barang, tunggu seluruh data dimuat, dan pilih alasan perubahan.');
      return;
    }
    if (!window.confirm('Seluruh perubahan data barang dan dokumen akan langsung disimpan serta dicatat pada timeline audit. Lanjutkan?')) event.preventDefault();
  });
})();
