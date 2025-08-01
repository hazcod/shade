function filterTable() {
	var input = document.getElementById("searchInput");
	var filter = input.value.toUpperCase();
	var table = document.getElementById("saasTable");
	var tr = table.getElementsByTagName("tr");
	for (var i = 1; i < tr.length; i++) {
		var td = tr[i].getElementsByTagName("td")[0];
		if (td) {
			var txtValue = td.textContent || td.innerText;
			if (txtValue.toUpperCase().indexOf(filter) > -1) {
				tr[i].style.display = "";
			} else {
				tr[i].style.display = "none";
			}
		}
	}
}