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
    select.form.submit();
}