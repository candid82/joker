function toggleTypes(e) {
  e.preventDefault(e);
  e.target.innerText = e.target.innerText === "hide types" ? "show types" : "hide types";
  e.target.parentNode.querySelectorAll('code').forEach(el => el.classList.toggle('hide'));
}

const terms = [{search-terms}];

const els = document.querySelectorAll('a.types');
els.forEach(el => el.addEventListener('click', toggleTypes));


(() => {

	let selectedIndex = 0;

	function onMouseOver(e) {
		const i = parseInt(e.target.dataset.index);
		if (!isNaN(i)) {
			updateSelection(i);
		}
	}

	function onInput(e) {
		searchSuggestionsEl.classList.add('hide');
		if (e.target.value.match(/^\s*$/)) {
			return;
		}
		let suggestions = [];
		for (i = 0; i < terms.length; i++) {
			if (terms[i].includes(e.target.value)) {
				if (suggestions.push(terms[i]) >= 20) break;
			}
		}
		if (suggestions.length != 0) {
			searchSuggestionsEl.innerHTML = suggestions.map((term, i) => '<li data-index="' + i + '">'+ term + '</li>').join('');
			searchSuggestionsEl.querySelectorAll('li').forEach(el => {
				el.addEventListener('mouseover', onMouseOver);
				el.addEventListener('click', submitSearch);
			});
			searchSuggestionsEl.classList.remove('hide');
			updateSelection(0);
		}
	}

	function updateSelection(newIndex) {
		let items = searchSuggestionsEl.querySelectorAll('li');
		if (newIndex < 0 || newIndex >= items.length) {
			return;
		}
		if (items[selectedIndex]) {
			items[selectedIndex].classList.remove('selected');
		}
		items[newIndex].classList.add('selected');
		selectedIndex = newIndex;
	}

	function submitSearch() {
		let items = searchSuggestionsEl.querySelectorAll('li');
		if (items.length == 0) {
			return;
		}
		const url = items[selectedIndex].innerText.replace('/', '.html#');
		document.location = url;
	}

	function onKeydown(e) {
		if (e.keyCode == 38 || (e.ctrlKey && e.key == 'p')) { // up arrow or CTRL+p
			e.preventDefault();
			updateSelection(selectedIndex - 1);
		} else if (e.keyCode == 40 || (e.ctrlKey && e.key == 'n')) { // down arrow or CTRL+n
			e.preventDefault();
			updateSelection(selectedIndex + 1);
		} else if (e.keyCode == 13) { // enter
			submitSearch();
		}
	}

	const searchSuggestionsEl = document.getElementById('search-suggestions');
	const searchInputEl = document.getElementById('search-input');
	searchInputEl.addEventListener('input', onInput);
	searchInputEl.addEventListener('keydown', onKeydown);
})();
