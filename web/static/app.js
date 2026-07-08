// Global JS — delegated interactions for components that appear many times per page.

document.addEventListener('click', function (event) {
	const button = event.target.closest('.copy-btn');
	if (!button) {
		return;
	}

	event.preventDefault();
	event.stopPropagation();

	const text = button.dataset.copy;
	if (!text) {
		return;
	}

	navigator.clipboard.writeText(text).then(function () {
		button.classList.add('is-copied');
		window.setTimeout(function () {
			button.classList.remove('is-copied');
		}, 1500);
	});
});
