document.querySelectorAll('.js-eval').forEach(el => {
    if (el.querySelector('script[src]')) {
      el.querySelectorAll('script[src]').forEach(s => {
        const n = document.createElement('script');
        n.src = s.src;
        document.body.appendChild(n);
      });

      el.style.display = 'none';
    } else {
      eval(el.textContent);
      el.style.display = 'none';
    }
  });