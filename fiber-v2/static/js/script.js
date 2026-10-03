// Sidebar active state handling for panel
document.addEventListener('DOMContentLoaded', () => {
    const path = window.location.pathname;

    // Clear any pre-set active classes
    document.querySelectorAll('.admin-sidebar a.nav-single, .admin-sidebar .nav-item').forEach(el => {
        el.classList.remove('active');
    });

    // Helper: mark active for matching links
    const markActive = (selector, matcher) => {
        const links = Array.from(document.querySelectorAll(selector));
        const match = links.find(a => matcher(a.getAttribute('href') || ''));
        if (match) {
            match.classList.add('active');
            // If inside a group, also expand header (optional if CSS supports it)
            const groupItems = match.closest('[data-group-items]');
            if (groupItems) {
                const group = groupItems.getAttribute('data-group-items');
                const header = document.querySelector(`.nav-group-header[data-group="${group}"]`);
                groupItems.classList.add('open');
                header && header.classList.add('open');
            }
        }
    };

    // Dashboard
    markActive('.admin-sidebar a.nav-single[href="/panel"]', href => path === '/panel');

    // Users section
    markActive('.admin-sidebar [data-group-items="users"] .nav-item', href => path.startsWith('/panel/kullanicilar') || path.startsWith('/panel/kullanici'));

    // Options (Seçenekler) section
    markActive('.admin-sidebar [data-group-items="settings"] .nav-item', href => path.startsWith('/panel/secenek'));

    // Other known groups can be added similarly as needed
});



// Mobil menü accordion - bir açılınca diğerleri kapansın
document.addEventListener('DOMContentLoaded', function() {
    document.addEventListener('click', function(e) {
        var btn = e.target.closest('.mobile-nav__container .main-menu__list .dropdown > a button');
        if (!btn) return;
        var clickedLi = btn.closest('li');
        document.querySelectorAll('.mobile-nav__container .main-menu__list > li').forEach(function(li) {
            if (li !== clickedLi) {
                var subUl = li.querySelector('ul');
                var liBtn = li.querySelector('button');
                if (subUl) { subUl.style.display = 'none'; }
                if (liBtn) liBtn.classList.remove('expanded');
                var a = li.querySelector('a');
                if (a) a.classList.remove('expanded');
            }
        });
    });
});


// Mobil: slider'ı header altına it
(function(){
    function fixSlider(){
        if(window.innerWidth > 991) return;
        var hf = document.querySelector('.header-four');
        var slider = document.querySelector('.main-slider-three');
        if(!slider) slider = document.querySelector('.main-slider-three__wrapper');
        if(!slider) slider = document.querySelector('section[class*="slider"]');
        if(hf && slider){
            slider.style.marginTop = hf.offsetHeight + 'px';
        }
    }
    document.addEventListener('DOMContentLoaded', fixSlider);
    window.addEventListener('resize', fixSlider);
    window.addEventListener('load', fixSlider);
})();

// Mobil: slider'ı header altına it
(function(){
    function fixSlider(){
        if(window.innerWidth > 991) return;
        var hf = document.querySelector('.header-four');
        var slider = document.querySelector('.main-slider-three');
        if(!slider) slider = document.querySelector('.main-slider-three__wrapper');
        if(!slider) slider = document.querySelector('section[class*="slider"]');
        if(hf && slider){
            slider.style.marginTop = hf.offsetHeight + 'px';
        }
    }
    document.addEventListener('DOMContentLoaded', fixSlider);
    window.addEventListener('resize', fixSlider);
    window.addEventListener('load', fixSlider);
})();

document.addEventListener('DOMContentLoaded', function () {
  var btn = document.querySelector('.mobile-nav__btn.mobile-nav__toggler');
  if (!btn) return;
  // P2-4: yeni header'da (.nv-hdr) menü düğmesi, Esc, odak ve durum
  // frontend-header-init.jet'te yönetiliyor; buradaki eski "hamburger"
  // sınıf eklemeleri (navbar-hamburger-btn / .bar) etiketi bozuyordu.
  if (btn.closest('.nv-hdr')) return;

  btn.classList.add('navbar-toggler','nav-btn-area','navbar-hamburger-btn','collapsed');

  var bars = btn.querySelectorAll('span');
  bars.forEach(function (b) { b.classList.add('bar'); });

  function syncState(){
    var open = document.body.classList.contains('mobile-nav-expanded');
    if (open) {
      btn.classList.remove('collapsed');
      btn.setAttribute('aria-expanded','true');
      btn.classList.remove('collapsed');
    } else {
      btn.classList.add('collapsed');
      btn.setAttribute('aria-expanded','false');
    }
  }

  syncState();
  btn.addEventListener('click', function(){ setTimeout(syncState, 0); });

  // --- Klavye erisimi ---
  // Hamburger artik <button>; Escape menuyu kapatir ve odagi butona
  // geri dondurur. Menu acilinca odak icindeki ilk baglantiya tasinir,
  // aksi halde klavye kullanicisi acilan menuye ulasamaz.
  var nvWrapper = document.getElementById('nvMobileNav');

  function nvCloseNav() {
    if (!document.body.classList.contains('mobile-nav-expanded')) return;
    if (nvWrapper) nvWrapper.classList.remove('expanded');
    document.body.classList.remove('mobile-nav-expanded', 'locked');
    syncState();
    btn.focus();
  }

  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape' || e.key === 'Esc') nvCloseNav();
  });

  btn.addEventListener('click', function () {
    setTimeout(function () {
      if (!document.body.classList.contains('mobile-nav-expanded')) return;
      if (!nvWrapper) return;
      var first = nvWrapper.querySelector('a, button');
      if (first) first.focus();
    }, 60);
  });
});


// Menü dışına tıklayınca kapat + kapanınca alt menüleri sıfırla
$(document).ready(function() {
    function resetSubmenus() {
        $('.mobile-nav__container .main-menu__list ul').hide();
        $('.mobile-nav__container .main-menu__list button').removeClass('expanded');
        $('.mobile-nav__container .main-menu__list a').removeClass('expanded');
    }

    // Dışarı tıklayınca kapat
    $(document).on('click', function(e) {
        if (!$('.mobile-nav__wrapper').hasClass('expanded')) return;
        if (!$(e.target).closest('.header-four, .mobile-nav__wrapper').length) {
            $('.mobile-nav__wrapper').removeClass('expanded');
            $('body').removeClass('mobile-nav-expanded locked');
            resetSubmenus();
        }
    });

    // Kapatma butonuna tıklayınca da sıfırla
    $(document).on('click', '.mobile-nav__toggler, .mobile-nav__close', function() {
        setTimeout(function() {
            if (!$('.mobile-nav__wrapper').hasClass('expanded')) {
                resetSubmenus();
            }
        }, 100);
    });
});
