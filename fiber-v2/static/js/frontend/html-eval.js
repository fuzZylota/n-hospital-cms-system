const el = document.querySelectorAll(".html-eval");
  
if (el) {
    for(let i = 0; i < el.length; i++) {
        // Mevcut metni al
        let raw = el[i].textContent || el[i].innerText;

        // Eğer başında/sonunda çift tırnak varsa onları temizle
        raw = raw.replace(/^"(.*)"$/, '$1');

        // Şimdi div'in içeriğini HTML olarak yeniden yaz
        el[i].innerHTML = raw;
    }
}