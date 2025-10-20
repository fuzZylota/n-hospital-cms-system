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