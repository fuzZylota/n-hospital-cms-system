(function () {
  'use strict';

  function selectAll(selector, root) {
    return Array.prototype.slice.call((root || document).querySelectorAll(selector));
  }

  function setOpen(acc, open) {
    var button = acc.querySelector('.doc-acc-header');
    var panel = acc.querySelector('.doc-acc-content');
    if (!button || !panel) return;
    acc.classList.toggle('open', open);
    button.setAttribute('aria-expanded', String(open));
    panel.hidden = !open;
  }

  function closeAllAccordions(except) {
    selectAll('.doc-page .doc-accordion').forEach(function (acc) {
      if (acc !== except) setOpen(acc, false);
    });
  }

  function initAccordions() {
    var accordions = selectAll('.doc-page .doc-accordion');
    var initial = document.querySelector('.doc-page .doc-accordion.open') || accordions[0];
    accordions.forEach(function (acc) {
      var button = acc.querySelector('.doc-acc-header');
      if (!button) return;
      setOpen(acc, acc === initial);
      button.addEventListener('click', function () {
        var isOpen = acc.classList.contains('open');
        if (isOpen) {
          setOpen(acc, false);
        } else {
          closeAllAccordions(acc);
          setOpen(acc, true);
        }
      });
      button.addEventListener('keydown', function (event) {
        if (event.key === 'Escape' && acc.classList.contains('open')) {
          setOpen(acc, false);
          event.preventDefault();
        }
      });
    });
  }

  document.addEventListener('DOMContentLoaded', initAccordions);
})();



