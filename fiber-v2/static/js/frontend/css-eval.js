document.querySelectorAll('.css-eval').forEach(el => {
    const style = document.createElement('style');
    style.textContent = el.textContent;
    document.head.appendChild(style);
    el.style.display = 'none';
  });