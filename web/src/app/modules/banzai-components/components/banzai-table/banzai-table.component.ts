import { ChangeDetectorRef, Component, EventEmitter, Input, NgZone, Output, ViewChild } from '@angular/core';
import { MatSort } from '@angular/material/sort';
import { MatTableDataSource } from '@angular/material/table';
import { TimeAgoPipe } from 'time-ago-pipe';
import { DatePipe, DecimalPipe } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { expandAndCollapseAnimation } from './animation/expand-and-collapse.animation';
import { TruncateAtMiddlePipe } from './pipe/truncate-at-middle.pipe';
import { CellType, PipeConfig, PipeType, TableData, TableDesignConfig, TableRowConfig, TableRowItem } from './model/tabledata';
import { ToFixedNumberPipe } from './pipe/to-fixed-number.pipe';

@Component({
  selector: 'app-banzai-table',
  templateUrl: './banzai-table.component.html',
  styleUrls: ['./banzai-table.component.scss'],
  animations: [expandAndCollapseAnimation],
})
export class BanzaiTableComponent {

  readonly nameKeyForQuery = 'name';

  private _sort: MatSort;

  @ViewChild(MatSort) set sort(value: MatSort) {
    this._sort = value;
    if (this._dataSource) {
      this._dataSource.sortingDataAccessor = (item: TableRowItem, property: string) => {
        return item.config[property].getValue();
      };

      this._dataSource.sort = this.sort;
    }
  }

  @Output() selectRowEvent: EventEmitter<{ row: number }> = new EventEmitter();

  @Input() isLoading: boolean;
  @Input() tableType: string; // TODO (colin): refactor this with `appStatusColor` type
  @Input() expandedDetails: boolean;
  @Input() emptyListText = 'There is no data in the table';
  @Input() filterExactMatch: boolean;

  @Input() set filterOnColumn(value: string) {
    this._filterOnColumn = value;
  }

  @Input() set changeFilter(filterValue: string) {
    this._filterValue = filterValue ? filterValue : '';
    if (this._dataSource) {
      this._dataSource.filter = this.filterValue;
    }
  }

  @Input() set dataSource(tableData: TableData) {
    if (tableData) {
      this.tableItems = tableData;
      this.tableStyle = tableData.designType;
      this.sortedTableData = this.tableItems.items.slice();
      if (!this._dataSource || !this._dataSource.data || this._dataSource.data.length === 0 || !tableData.stopReRender) {
        this._dataSource = new MatTableDataSource<TableRowItem>(this.sortedTableData);
        this.overrideFilter();
        this._dataSource.filter = this.filterValue;
      }
      this.processQueries();
    } else {
      this.tableItems = null;
      this.sortedTableData = null;
      this._dataSource = null;
    }
  }

  timeAgoPipe: TimeAgoPipe;
  cellType = CellType;
  _dataSource = new MatTableDataSource<TableRowItem>();
  tableItems: TableData;
  sortedTableData: TableRowItem[];
  private _filterValue: string;
  private _filterOnColumn: string;
  expandedElement: TableRowItem;
  nameFromQuery: string;
  tableStyle: TableDesignConfig;

  constructor(
    private datePipe: DatePipe,
    private decimalPipe: DecimalPipe,
    private truncateAtMiddlePipe: TruncateAtMiddlePipe,
    private toFixedNumber: ToFixedNumberPipe,
    private changeDetectorRef: ChangeDetectorRef,
    private zone: NgZone,
    private router: Router,
    private activatedRoute: ActivatedRoute,
  ) {
    this.timeAgoPipe = new TimeAgoPipe(changeDetectorRef, zone);
    this.nameFromQuery = this.activatedRoute.snapshot.queryParamMap.get(this.nameKeyForQuery);
  }

  get headers(): string[] {
    return Object.keys(this.tableItems.headers);
  }

  get filterValue(): string {
    return this._filterValue;
  }

  get sort(): MatSort {
    return this._sort;
  }

  get filterOnColumn(): string {
    return this._filterOnColumn;
  }

  getText(row: TableRowConfig): string {
    if (row.display) {

      const text = row.getValue();
      if (row.display.pipeConfig) {
        const value = this.transformTextByPipe(text.toString(), row.display.pipeConfig);
        if (row.display.postFix) {
          return `${value} ${row.display.postFix}`;
        } else {
          return value;
        }
      }

      return text.toString();

    } else {
      return '';
    }
  }

  private transformTextByPipe(text: string, pipeConfig: PipeConfig): string {
    switch (pipeConfig.pipe) {
      case PipeType.TimeAgo:
        return this.timeAgoPipe.transform(text.toString());
      case PipeType.Date:
        return this.datePipe.transform(text.toString(), pipeConfig.format);
      case PipeType.Number:
        if (!isNaN(Number(text))) {
          return this.decimalPipe.transform(text.toString(), pipeConfig.format);
        } else {
          return text;
        }
      case PipeType.TruncateAtMiddle:
        return this.truncateAtMiddlePipe.transform(text.toString(), pipeConfig.format);
      case PipeType.ToFixedNumber:
        return this.toFixedNumber.transform(text.toString(), pipeConfig.format);
      default:
        return text;
    }
  }

  getTooltip(row: TableRowConfig): string {

    const config = row.tooltipConfig;
    if (config) {
      const tooltip = config.tooltip;
      const tooltipPipeConfig = config.pipeConfig;
      if (tooltipPipeConfig) {
        return this.transformTextByPipe(tooltip, tooltipPipeConfig);
      } else {
        return tooltip;
      }

    }

    return '';
  }

  expandDetails(row: TableRowItem) {
    if (this.expandedElement === row) {
      this.expandedElement = null;
    } else {
      this.expandedElement = row;
    }

    this.changeQueryParam();

  }

  clickRowEvent(row: TableRowItem) {
    const index = this.getRowIndex(row);
    if (this.selectRowEvent) {
      this.selectRowEvent.emit({ row: index });
    }
  }

  getRowIndex(item: TableRowItem): number {
    return item.index;
  }

  getGroupName(item: TableRowItem): string {
    return item.groupName;
  }

  private processQueries() {
    if (this.nameFromQuery) {
      this.expandedElement = this.sortedTableData.find(d => {
        if (d.config[this.nameKeyForQuery]) {
          return d.config[this.nameKeyForQuery].getValue() === this.nameFromQuery;
        } else {
          return false;
        }
      });
    }
  }

  private changeQueryParam() {
    let name: string = null;
    if (this.expandedElement && this.expandedElement.config[this.nameKeyForQuery]) {
      name = this.expandedElement.config[this.nameKeyForQuery].getValue();
    }

    this.router.navigate(['.'], {
      relativeTo: this.activatedRoute,
      queryParams: { name: name },
      queryParamsHandling: 'merge',
    });
  }

  getModelToDetails(row: TableRowItem): { index: number, isOpen: boolean } {
    const index = this.getRowIndex(row);
    const isOpen = row === this.expandedElement;
    return { index: index, isOpen: isOpen };
  }

  showTable(): boolean {
    return !this.isLoading && this._dataSource && this._dataSource.data.length !== 0;
  }

  private overrideFilter() {
    this._dataSource.filterPredicate = (item: TableRowItem, filter: string) => {
      const keys = Object.keys(item.config);
      return keys.some((key) => {
        const value = `${item.config[key].getValue()}`;
        if (this.filterOnColumn) {
          if (this.filterOnColumn === key) {
            return this.isFilterMatch(value, filter);
          } else {
            return false;
          }
        } else {
          return this.isFilterMatch(value, filter);
        }
      });
    };
  }

  private isFilterMatch(value: string, commaSeparatedMultipleFilter: string): boolean {
    const filterValueSeparator = ',';
    const filters = commaSeparatedMultipleFilter.toLowerCase().split(filterValueSeparator);
    value = value.toLowerCase();
    const values = value.split(filterValueSeparator);

    if (this.filterExactMatch) {
      // filter only exact matches, whole words, like tags
      for (const filter of filters) {
        if (!values.includes(filter)) {
          return false;
        }
      }
    } else {
      // filter partial matches
      for (const filter of filters) {
        if (!value.includes(filter)) {
          return false;
        }
      }
    }

    return true;
  }

}
