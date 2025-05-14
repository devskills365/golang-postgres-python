const draggableItems = document.querySelectorAll('.draggable-item');
const droppableAreaRows = document.getElementById('droppable-area-rows');
const droppableAreaCols = document.getElementById('droppable-area-cols');
const initialList = document.getElementById('initial-list');
const tableContainer = document.getElementById('table-container');
const filterContainer = document.getElementById('filter-container');

// Récupérer df_filtered_id depuis le HTML
const dfFilteredId = document.currentScript.getAttribute('data-df-filtered-id') || '{{ .df_filtered_id }}';

let rowColumns = [];
let colColumns = [];
let tableData = [];
let filteredTableData = [];

draggableItems.forEach(item => {
    item.addEventListener('dragstart', handleDragStart);
});

droppableAreaRows.addEventListener('dragover', handleDragOver);
droppableAreaRows.addEventListener('drop', event => handleDrop(event, 'row'));

droppableAreaCols.addEventListener('dragover', handleDragOver);
droppableAreaCols.addEventListener('drop', event => handleDrop(event, 'col'));

initialList.addEventListener('dragover', handleDragOver);
initialList.addEventListener('drop', event => handleDrop(event, 'initial'));

function handleDragStart(event) {
    event.dataTransfer.setData('text/plain', event.target.dataset.column);
    event.dataTransfer.setData('source-id', event.target.id);
}

function handleDragOver(event) {
    event.preventDefault();
}

function handleDrop(event, type) {
    event.preventDefault();
    const column = event.dataTransfer.getData('text/plain');
    const sourceId = event.dataTransfer.getData('source-id');
    const draggedElement = document.querySelector(`[data-column="${column}"][id="${sourceId}"]`) || document.querySelector(`[data-column="${column}"]`);

    if (!draggedElement) return;

    if (draggedElement.parentElement) {
        draggedElement.parentElement.removeChild(draggedElement);
    }

    if (rowColumns.includes(column)) {
        rowColumns.splice(rowColumns.indexOf(column), 1);
    } else if (colColumns.includes(column)) {
        colColumns.splice(colColumns.indexOf(column), 1);
    }

    if (type === 'row' && !rowColumns.includes(column)) {
        rowColumns.push(column);
        addColumnToArea(column, droppableAreaRows, rowColumns, type);
    } else if (type === 'col' && !colColumns.includes(column)) {
        colColumns.push(column);
        addColumnToArea(column, droppableAreaCols, colColumns, type);
    } else if (type === 'initial') {
        addColumnToArea(column, initialList, null, type);
    }

    togglePlaceholders();
    sendColumnsToServer();
}

function addColumnToArea(column, area, columnList, type) {
    const newItem = document.createElement('div');
    newItem.classList.add('draggable-item');
    newItem.textContent = column;
    newItem.setAttribute('draggable', 'true');
    newItem.setAttribute('data-column', column);
    newItem.id = `drag-${column}-${Date.now()}`;
    newItem.addEventListener('dragstart', handleDragStart);

    if (type !== 'initial') {
        newItem.addEventListener('click', function () {
            area.removeChild(newItem);
            if (columnList) {
                columnList.splice(columnList.indexOf(column), 1);
            }
            togglePlaceholders();
            sendColumnsToServer();
        });
    }

    area.appendChild(newItem);
}

function togglePlaceholders() {
    const rowPlaceholder = droppableAreaRows.querySelector('#placeholder-rows');
    const colPlaceholder = droppableAreaCols.querySelector('#placeholder-cols');
    rowPlaceholder.style.display = droppableAreaRows.children.length > 1 ? 'none' : 'block';
    colPlaceholder.style.display = droppableAreaCols.children.length > 1 ? 'none' : 'block';
}

function sendColumnsToServer() {
    if (!dfFilteredId) {
        console.error('Erreur : df_filtered_id non défini');
        alert('Erreur : Impossible de charger les données. Veuillez recharger la page.');
        return;
    }

    fetch('/process_columns', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({
            row_columns: rowColumns,
            col_columns: colColumns,
            value_column: 'Valeur',
            session_id: dfFilteredId // Envoyer l'identifiant de session
        })
    })
    .then(response => {
        if (!response.ok) {
            throw new Error(`Erreur HTTP ${response.status}: ${response.statusText}`);
        }
        return response.json();
    })
    .then(data => {
        tableData = data.data.map(row => {
            const rowData = {};
            data.columns.forEach((col, index) => {
                rowData[col.join(' ')] = row[index];
            });
            return rowData;
        });

        tableData.columns = data.columns;
        filteredTableData = tableData;
        generateTable(data);
        generateFilters();
    })
    .catch(error => {
        console.error('Erreur lors de la requête AJAX:', error);
        alert('Erreur lors de la génération du tableau. Veuillez réessayer.');
    });
}

function generateTable(data) {
    tableContainer.innerHTML = '';

    const table = document.createElement('table');
    const thead = document.createElement('thead');
    const tbody = document.createElement('tbody');

    const columns = data.columns;
    const levels = columns.length > 0 ? columns[0].length : 0;

    for (let level = 0; level < levels; level++) {
        const headerRow = document.createElement('tr');
        let previousValue = null;
        let colspan = 0;

        columns.forEach((col, index) => {
            const currentValue = col[level];

            if (currentValue === previousValue) {
                colspan += 1;
            } else {
                if (colspan > 0) {
                    headerRow.lastChild.setAttribute('colspan', colspan);
                }
                const th = document.createElement('th');
                th.textContent = currentValue || '';
                headerRow.appendChild(th);
                previousValue = currentValue;
                colspan = 1;
            }

            if (index === columns.length - 1 && colspan > 1) {
                headerRow.lastChild.setAttribute('colspan', colspan);
            }
        });
        thead.appendChild(headerRow);
    }

    data.data.forEach(row => {
        const tr = document.createElement('tr');
        columns.forEach((col, index) => {
            const colKey = col.join(' ');
            const td = document.createElement('td');
            td.textContent = row[index] || ' ';
            tr.appendChild(td);
        });
        tbody.appendChild(tr);
    });

    table.appendChild(thead);
    table.appendChild(tbody);
    tableContainer.appendChild(table);
    mergeTableCells();
}

const lastValidValues = {};

function generateFilters() {
    filterContainer.innerHTML = '';

    const allRows = [...rowColumns];
    const allColumns = [...colColumns];

    console.log("Columns for filters:", allRows);
    console.log('Columns for no filter:', allColumns);

    allRows.forEach(col => {
        if (colColumns.includes(col)) {
            return;
        }

        const colKey = Array.isArray(col) ? col.join(' ') : col;
        let uniqueValues = [...new Set(tableData.map(row => row[colKey]))].filter(val => val !== undefined);

        if (uniqueValues.length === 0) {
            uniqueValues = lastValidValues[colKey] || ["Valeur manquante"];
        } else {
            lastValidValues[colKey] = uniqueValues;
        }

        console.log(`Unique values for ${colKey}:`, uniqueValues);

        const filterGroup = document.createElement('div');
        filterGroup.classList.add('filter-group');

        const filterTitle = document.createElement('div');
        filterTitle.classList.add('filter-title');
        filterTitle.innerHTML = `<span class="icon-orange">+</span> Filtrer sur ${col}`;
        filterTitle.style.cursor = 'pointer';

        const checkboxContainer = document.createElement('div');
        checkboxContainer.classList.add('checkbox-container');
        checkboxContainer.style.display = 'none';

        filterTitle.addEventListener('click', () => {
            checkboxContainer.style.display = checkboxContainer.style.display === 'none' ? 'block' : 'none';
        });

        uniqueValues.forEach(value => {
            const checkboxWrapper = document.createElement('div');
            const checkbox = document.createElement('input');
            checkbox.type = 'checkbox';
            checkbox.value = value;
            checkbox.setAttribute('data-column', colKey);

            const checkboxLabel = document.createElement('label');
            checkboxLabel.textContent = value;

            checkbox.addEventListener('change', applyFilters);

            checkboxWrapper.appendChild(checkbox);
            checkboxWrapper.appendChild(checkboxLabel);
            checkboxContainer.appendChild(checkboxWrapper);
        });

        filterGroup.appendChild(filterTitle);
        filterGroup.appendChild(checkboxContainer);
        filterContainer.appendChild(filterGroup);
    });
}

function doesRowMatchFilters(row, filters) {
    return Object.keys(filters).every(column => {
        const filterValues = filters[column];
        return Object.values(row).some(rowValue => filterValues.includes(rowValue));
    });
}

function applyFilters() {
    filteredTableData = [...tableData];
    console.log("Applying filters...");

    const checkedCheckboxes = filterContainer.querySelectorAll('input[type="checkbox"]:checked');
    const filters = {};

    checkedCheckboxes.forEach(checkbox => {
        const column = checkbox.getAttribute('data-column');
        const value = checkbox.value;

        if (!filters[column]) {
            filters[column] = [];
        }
        filters[column].push(value);
    });

    filteredTableData = filteredTableData.filter(row => doesRowMatchFilters(row, filters));

    console.log('Tableau filtré:', filteredTableData);

    generateTable({
        columns: tableData.columns,
        data: filteredTableData.map(row => {
            return tableData.columns.map(col => {
                const key = Array.isArray(col) ? col.join(' ') : col;
                return row[key];
            });
        })
    });
}

function mergeTableCells() {
    const table = document.querySelector('#table-container table');
    if (!table) return;

    const rows = table.rows;
    const rowCount = rows.length;

    for (let col = 0; col < rows[0].cells.length; col++) {
        let startRow = 0;
        let value = rows[0].cells[col].innerText;
        for (let row = 1; row <= rowCount; row++) {
            if (row < rowCount && rows[row].cells[col].innerText === value) {
                continue;
            } else {
                if (row - startRow > 1) {
                    rows[startRow].cells[col].rowSpan = row - startRow;
                    for (let i = startRow + 1; i < row; i++) {
                        rows[i].cells[col].style.display = 'none';
                    }
                }
                if (row < rowCount) {
                    startRow = row;
                    value = rows[row].cells[col].innerText;
                }
            }
        }
    }
}

document.getElementById('download-xlsx').addEventListener('click', downloadXLSX);
document.getElementById('download-csv').addEventListener('click', downloadCSV);
document.getElementById('download-pdf').addEventListener('click', downloadPDF);

function downloadXLSX() {
    if (!filteredTableData || filteredTableData.length === 0) {
        alert("Aucune donnée sélectionnée.");
        return;
    }

    fetch(`/get_data_temp?session_id=${encodeURIComponent(dfFilteredId)}`)
        .then(response => {
            if (!response.ok) {
                throw new Error(`Erreur HTTP ${response.status}: ${response.statusText}`);
            }
            return response.json();
        })
        .then(data => {
            const wb = XLSX.utils.book_new();
            const ws = XLSX.utils.json_to_sheet(data);
            XLSX.utils.book_append_sheet(wb, ws, 'Données');
            XLSX.writeFile(wb, 'donnees_filtrees.xlsx');
        })
        .catch(error => {
            console.error('Erreur téléchargement XLSX:', error);
            alert('Erreur lors du téléchargement XLSX. Veuillez réessayer.');
        });
}

function downloadCSV() {
    if (!filteredTableData || filteredTableData.length === 0) {
        alert("Aucune donnée sélectionnée.");
        return;
    }

    fetch(`/get_data_temp?session_id=${encodeURIComponent(dfFilteredId)}`)
        .then(response => {
            if (!response.ok) {
                throw new Error(`Erreur HTTP ${response.status}: ${response.statusText}`);
            }
            return response.json();
        })
        .then(data => {
            const csvContent = [
                Object.keys(data[0]).join(','),
                ...data.map(row => Object.values(row).map(value => {
                    if (typeof value === 'string' && (value.includes(',') || value.includes('"') || value.includes('\n'))) {
                        return `"${value.replace(/"/g, '""')}"`;
                    }
                    return value;
                }).join(','))
            ].join('\n');

            const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
            const url = URL.createObjectURL(blob);
            const link = document.createElement('a');
            link.setAttribute('href', url);
            link.setAttribute('download', 'donnees_filtrees.csv');
            document.body.appendChild(link);
            link.click();
            document.body.removeChild(link);
        })
        .catch(error => {
            console.error('Erreur téléchargement CSV:', error);
            alert('Erreur lors du téléchargement CSV. Veuillez réessayer.');
        });
}

function downloadPDF() {
    if (!filteredTableData || filteredTableData.length === 0) {
        alert("Aucune donnée sélectionnée.");
        return;
    }

    fetch(`/get_data_temp?session_id=${encodeURIComponent(dfFilteredId)}`)
        .then(response => {
            if (!response.ok) {
                throw new Error(`Erreur HTTP ${response.status}: ${response.statusText}`);
            }
            return response.json();
        })
        .then(data => {
            const { jsPDF } = window.jspdf;
            const doc = new jsPDF({ orientation: 'portrait' });

            const pdfTitleElement = document.getElementById('pdf-title');
            const pdfTitle = pdfTitleElement ? pdfTitleElement.textContent.trim() : 'Données Filtrées';
            doc.setFontSize(16);
            doc.text(pdfTitle, 10, 15);

            const columns = Object.keys(data[0]);
            const bodyRows = data.map(row => Object.values(row));

            doc.autoTable({
                startY: 25,
                head: [columns],
                body: bodyRows,
                theme: 'grid',
                headStyles: { fillColor: [0, 107, 69], textColor: [255, 153, 0], fontStyle: 'bold' },
                bodyStyles: { fontSize: 10, cellPadding: 2, halign: 'center' },
                alternateRowStyles: { fillColor: [245, 245, 245] },
                margin: { left: 10, right: 10 },
                styles: { overflow: 'linebreak', cellWidth: 'auto' }
            });

            doc.save('donnees_filtrees.pdf');
        })
        .catch(error => {
            console.error('Erreur téléchargement PDF:', error);
            alert('Erreur lors du téléchargement PDF. Veuillez réessayer.');
        });
}

