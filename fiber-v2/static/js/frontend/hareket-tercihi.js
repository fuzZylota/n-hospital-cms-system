/*
 * Hareket tercihi ve slider duraklatma.
 *
 * 1) Isletim sisteminde "hareketi azalt" secili kullanicilarda otomatik
 *    donen tum carousel'ler durdurulur. CSS animasyonlari ayrica
 *    style.css icindeki prefers-reduced-motion blogu ile kapatiliyor;
 *    burasi JS ile surulen otomatik gecisleri kapsiyor.
 * 2) Anasayfa banner'i icin duraklat/devam butonu. WCAG 2.2.2 uyarinca
 *    5 saniyeden uzun suren otomatik hareketin durdurulabilmesi gerekir.
 */
(function () {
    'use strict';

    var LBL_DURDUR = 'Otomatik gecisi durdur';
    var LBL_DEVAM = 'Otomatik gecise devam et';
    var IKON_DURAKLAT = '\u2589\u2589';
    var IKON_OYNAT = '\u25B6';

    function reduceMotion() {
        return window.matchMedia &&
               window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    }

    // Slick (banner) ve Owl (doktorlar, merkezler, yorumlar) icin ortak
    // durdurma. Ilgili kutuphane yuklu degilse sessizce atlanir.
    function tumCarouselleriDurdur() {
        if (typeof window.jQuery === 'undefined') return;
        var $ = window.jQuery;

        $('.mediox-slick__carousel').each(function () {
            try { $(this).slick('slickPause'); } catch (e) {}
        });

        $('.mediox-owl__carousel, .owl-carousel').each(function () {
            try { $(this).trigger('stop.owl.autoplay'); } catch (e) {}
        });
    }

    function bannerDurdur(durdur) {
        if (typeof window.jQuery === 'undefined') return;
        var $ = window.jQuery;
        var $banner = $('.main-slider-three__carousel');
        if (!$banner.length) return;
        try {
            $banner.slick(durdur ? 'slickPause' : 'slickPlay');
        } catch (e) {}
    }

    function butonuKur() {
        var btn = document.getElementById('nvSliderPause');
        if (!btn) return;

        var ikon = btn.querySelector('.nv-slider-pause__icon');
        var duraklatildi = false;

        function durumUygula(yeni) {
            duraklatildi = yeni;
            btn.setAttribute('aria-pressed', duraklatildi ? 'true' : 'false');
            btn.setAttribute('aria-label', duraklatildi ? LBL_DEVAM : LBL_DURDUR);
            if (ikon) ikon.textContent = duraklatildi ? IKON_OYNAT : IKON_DURAKLAT;
            bannerDurdur(duraklatildi);
        }

        // Hareketi azalt tercihi aciksa buton bastan duraklatilmis baslar.
        if (reduceMotion()) durumUygula(true);

        btn.addEventListener('click', function () {
            durumUygula(!duraklatildi);
        });
    }

    function baslat() {
        // Carousel'ler tema JS'i tarafindan kuruluyor; kurulum bitsin diye
        // kisa bir gecikme birakiliyor.
        setTimeout(function () {
            if (reduceMotion()) tumCarouselleriDurdur();
            butonuKur();
        }, 300);
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', baslat);
    } else {
        baslat();
    }
})();
