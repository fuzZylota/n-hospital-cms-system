(function() {
    'use strict';
    var tabs = document.querySelectorAll('.sube-tab');
    var panels = document.querySelectorAll('.sube-panel');
    var tabsNav = document.getElementById('subeTabs');
    var arrowLeft = document.getElementById('subeTabsArrowLeft');
    var arrowRight = document.getElementById('subeTabsArrowRight');

    tabs.forEach(function(tab) {
        tab.addEventListener('click', function() {
            var target = this.getAttribute('data-tab');
            tabs.forEach(function(t) { t.classList.remove('active'); });
            panels.forEach(function(p) { p.classList.remove('active'); });
            this.classList.add('active');
            var panel = document.getElementById('tab-' + target);
            if (panel) panel.classList.add('active');
            history.replaceState(null, '', '#' + target);
            this.scrollIntoView({ behavior: 'smooth', inline: 'center', block: 'nearest' });
        });
    });
    var hash = window.location.hash.replace('#', '');
    if (hash) {
        var targetTab = document.querySelector('.sube-tab[data-tab="' + hash + '"]');
        if (targetTab) targetTab.click();
    }

    // Mobilde yatay sekme çubuğunun kaydırılabilir olduğunu belli eden
    // ok butonları ve ilk ziyarette küçük bir "kaydırılabilir" ipucu animasyonu.
    if (tabsNav && arrowLeft && arrowRight) {
        function updateArrows() {
            var maxScroll = tabsNav.scrollWidth - tabsNav.clientWidth;
            if (maxScroll <= 4) {
                arrowLeft.classList.remove('is-visible');
                arrowRight.classList.remove('is-visible');
                return;
            }
            arrowLeft.classList.toggle('is-visible', tabsNav.scrollLeft > 4);
            arrowRight.classList.toggle('is-visible', tabsNav.scrollLeft < maxScroll - 4);
        }

        arrowLeft.addEventListener('click', function() {
            tabsNav.scrollBy({ left: -140, behavior: 'smooth' });
        });
        arrowRight.addEventListener('click', function() {
            tabsNav.scrollBy({ left: 140, behavior: 'smooth' });
        });

        tabsNav.addEventListener('scroll', updateArrows);
        window.addEventListener('resize', updateArrows);
        updateArrows();

        // Sayfa ilk açıldığında, kaydırılabilir içerik varsa kısa bir
        // "nudge" (dürtme) animasyonu ile bunu kullanıcıya gösterir.
        window.addEventListener('load', function() {
            setTimeout(function() {
                var maxScroll = tabsNav.scrollWidth - tabsNav.clientWidth;
                if (maxScroll > 4 && tabsNav.scrollLeft <= 4) {
                    tabsNav.scrollTo({ left: 36, behavior: 'smooth' });
                    setTimeout(function() {
                        tabsNav.scrollTo({ left: 0, behavior: 'smooth' });
                    }, 550);
                }
            }, 700);
        });
    }
})();

// LİGHTBOX
function openLightbox(src, caption) {
    var existing = document.getElementById('fgLightbox');
    if (existing) existing.remove();

    var lb = document.createElement('div');
    lb.id = 'fgLightbox';
    lb.className = 'fg-lightbox';

    var closeBtn = document.createElement('button');
    closeBtn.className = 'fg-lightbox__close';
    closeBtn.innerHTML = '<i class="fas fa-times"></i>';
    closeBtn.onclick = function() { lb.remove(); };

    var img = document.createElement('img');
    img.className = 'fg-lightbox__img';
    img.src = src;
    img.alt = caption || '';

    lb.appendChild(img);
    if (caption) {
        var cap = document.createElement('div');
        cap.className = 'fg-lightbox__caption';
        cap.textContent = caption;
        lb.appendChild(cap);
    }
    lb.appendChild(closeBtn);
    lb.addEventListener('click', function(e) { if (e.target === lb) lb.remove(); });
    document.body.appendChild(lb);

    document.addEventListener('keydown', function esc(e) {
        if (e.key === 'Escape') { lb.remove(); document.removeEventListener('keydown', esc); }
    });
}
