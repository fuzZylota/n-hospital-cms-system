(function () {
function init() {
	var header = document.querySelectorAll('header.main-header');
	if (!header) return;

	function updateHeaderTransparency() {
		var scrolled = window.scrollY || window.pageYOffset || 0;
		if (scrolled > 0) {
            for(let i = 0; i < header.length; i++) {
                if(i === 0) continue;
				header[i].classList.remove('transparent-bg-on-homepage');
				header[i].classList.add('not-transparent-bg-on-homepage');
			}
		} else {
			for(let i = 0; i < header.length; i++) {
                if(i === 0) continue;
				header[i].classList.add('transparent-bg-on-homepage');
				header[i].classList.remove('not-transparent-bg-on-homepage');
			}
		}
	}

	// Initialize on load
	updateHeaderTransparency();

	// Listen for scroll and resize
	window.addEventListener('scroll', updateHeaderTransparency, { passive: true });
	window.addEventListener('resize', updateHeaderTransparency);
}

if (document.readyState === 'loading') {
	document.addEventListener('DOMContentLoaded', init);
} else {
	init();
}
})();
