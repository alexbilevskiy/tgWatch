function addRow(id, hasTitleColumn = true) {
    let lastRowSelector = id + ' tr:last';
    let lastInputSelector = id + ' tr:last input';
    let minusButtonSelector = id + ' tr .btn:first';
    let lastRow = $(lastRowSelector);
    let lastInput = $(lastInputSelector);
    let minusButton = $(minusButtonSelector);

    if (lastInput.val() === '') {
        alert('Empty input');
        return;
    }
    let newRow = lastRow.clone(false);
    lastRow.after(newRow);
    if(hasTitleColumn) {
        lastRow.children()[0].innerHTML = '(tbd)';
        lastRow.children()[2].innerHTML = minusButton.wrap('<div>').parent().html();
    } else {
        lastRow.children()[1].innerHTML = minusButton.wrap('<div>').parent().html();
    }


    let newLastInput = $(lastInputSelector);
    newLastInput.val('');
}

function deleteRow(row) {
    $(row).parents('tr').html('');
}

function changePhone(select) {
    document.cookie = 'acc=' + select.value + '; path=/';
    location.reload();
}

let Table = {
    tbody: $('#overview_table tbody'),
    head_rows: $('#overview_table thead .sort_by'),
    rows: $('#overview_table tbody tr'),
    sortable: $('#overview_table .sort_by')
};

let TableHandler = {
    lastSorted: null,

    sortBy: function (sortBy) {
        let rows = [].slice.call(Table.rows);

        if (this.lastSorted === sortBy) {
            rows.sort(function (a, b) {
                return parseFloat(a.dataset[sortBy]) - parseFloat(b.dataset[sortBy]);
            });
            this.lastSorted = null;
        } else {
            rows.sort(function (a, b) {
                return parseFloat(b.dataset[sortBy]) - parseFloat(a.dataset[sortBy]);
            });
            this.lastSorted = sortBy;
        }

        Table.tbody.html('').append(rows);
    },

    showActiveSortParam: function (sortBy) {
        $.each(Table.head_rows, function (key, row) {
            if (row.dataset.sortBy === sortBy) {
                row.classList.add('bg-success')
            } else {
                row.classList.remove('bg-success');
            }
        });
    }
};
Table.sortable.on('click', function (e) {
    TableHandler.showActiveSortParam(e.target.dataset.sortBy);
    TableHandler.sortBy(e.target.dataset.sortBy);
});

