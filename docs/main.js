function toggleTypes(e) {
  e.preventDefault(e);
  e.target.innerText = e.target.innerText === "hide types" ? "show types" : "hide types";
  e.target.parentNode.querySelectorAll('code').forEach(el => el.classList.toggle('hide'))
}

const els = document.querySelectorAll('a.types');
els.forEach(el => el.addEventListener('click', toggleTypes));
