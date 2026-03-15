const btn = document.getElementById("menuBtn");
const nav = document.getElementById("nav");
if (btn && nav) {
  btn.addEventListener("click", () => nav.classList.toggle("open"));
}