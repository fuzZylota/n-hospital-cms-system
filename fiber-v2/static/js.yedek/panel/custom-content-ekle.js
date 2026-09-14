// fiber-v2/static/js/panel/custom-content-ekle.js

(() => {
  "use strict";

  function $(id) {
    return document.getElementById(id);
  }

  function normalizeBoolCheckbox(checkboxEl, formData, fieldName) {
    // If unchecked, FormData does not include it; backend expects bool (false).
    // We force it explicitly.
    if (!checkboxEl) return;
    if (checkboxEl.checked) {
      formData.set(fieldName, "on");
    } else {
      formData.delete(fieldName);
      // Fiber BodyParser for bool usually treats missing as false, but we keep consistent:
      // Some implementations expect "false".
      formData.set(fieldName, "false");
    }
  }

  function applyUlasimMapping(formData) {
    const ctEl = $("content_type");
    const urlEl = $("url_name");
    const routeEl = $("route");

    if (!ctEl) return;

    // UI: ulasim, DB: other
    if (ctEl.value === "ulasim") {
      // Ensure URL + route
      if (urlEl && (!urlEl.value || urlEl.value.trim() === "")) urlEl.value = "ulasim";
      if (routeEl && (!routeEl.value || routeEl.value.trim() === "")) routeEl.value = "/ulasim";

      // Update formData too (in case values were empty before FormData created)
      formData.set("url_name", "ulasim");
      formData.set("route", "/ulasim");

      // IMPORTANT: prevent DB constraint / sorting function errors
      // // formData.set("content_type", "other");
    }
  }

  async function postForm(formEl) {
    const submitBtn = $("submitBtn") || formEl.querySelector('button[type="submit"]');

    try {
      if (submitBtn) submitBtn.disabled = true;

      const formData = new FormData(formEl);

      // Fix checkbox boolean
      normalizeBoolCheckbox($("is_active"), formData, "is_active");

      // Special mapping for Ulasim
      applyUlasimMapping(formData);

      const res = await fetch("/backend/add-custom-content", {
        method: "POST",
        body: formData,
        // credentials ensures cookies/session are sent
        credentials: "same-origin",
      });

      // The backend returns Redirect (/panel/ozel-icerikler/{ccid})
      // fetch follows redirects; res.url becomes final URL.
      if (res.ok) {
        // If it ended up on a panel page, go there
        if (res.url && res.url.includes("/panel/")) {
          window.location.href = res.url;
          return;
        }

        // If server did not redirect for some reason, fallback to reload
        window.location.reload();
        return;
      }

      // If backend returned HTML with error, show it
      const text = await res.text();
      console.error("Server error response:", res.status, text);
      alert("Hata! Sunucu istegi basarisiz oldu. (HTTP " + res.status + ")");
    } catch (err) {
      console.error(err);
      alert("Hata! Server Hatası: Lutfen daha sonra tekrar deneyin.");
    } finally {
      if (submitBtn) submitBtn.disabled = false;
    }
  }

  document.addEventListener("DOMContentLoaded", () => {
    // Try common form ids (if yours differs, it still falls back to the first form)
    const formEl =
      document.getElementById("customContentForm") ||
      document.getElementById("customContentCreateForm") ||
      document.querySelector("form");

    if (!formEl) return;

    formEl.addEventListener("submit", (e) => {
      e.preventDefault();
      postForm(formEl);
    });
  });
})();