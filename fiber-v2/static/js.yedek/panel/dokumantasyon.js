(function () {
  'use strict';

  function selectAll(selector, root) {
    return Array.prototype.slice.call((root || document).querySelectorAll(selector));
  }

  function onClick(el, handler) {
    el.addEventListener('click', handler, false);
  }

  function closeAllAccordions(except) {
    selectAll('.doc-accordion').forEach(function (acc) {
      if (acc !== except) acc.classList.remove('open');
    });
  }

  function initAccordions() {
    selectAll('.doc-accordion').forEach(function (acc) {
      var header = acc.querySelector('.doc-acc-header');
      if (!header) return;
      onClick(header, function () {
        var isOpen = acc.classList.contains('open');
        if (isOpen) {
          // If the clicked accordion is already open, close it (toggle off)
          acc.classList.remove('open');
        } else {
          // Otherwise, close others and open this one (single-open behavior)
          closeAllAccordions(acc);
          acc.classList.add('open');
        }
      });
    });
    // ensure one open by default (first)
    var first = document.querySelector('.doc-accordion');
    if (first && !document.querySelector('.doc-accordion.open')) {
      first.classList.add('open');
    }
  }

  document.addEventListener('DOMContentLoaded', initAccordions);
})();



