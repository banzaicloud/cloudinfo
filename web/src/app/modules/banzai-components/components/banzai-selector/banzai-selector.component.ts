import { Component, EventEmitter, Input, OnInit, Output } from '@angular/core';
import { SelectorGroup, SelectorItem } from './model/selector-item';
import { ActivatedRoute, Router } from '@angular/router';

@Component({
  selector: 'app-banzai-selector',
  templateUrl: './banzai-selector.component.html',
  styleUrls: ['./banzai-selector.component.scss'],
})
export class BanzaiSelectorComponent implements OnInit {

  private _groups: SelectorGroup[];
  private queryGroupValue: string;
  private queryItemValue: string;

  @Input() isLoading: boolean;
  @Input() isGroupSelect: boolean;
  @Input() selectorLabel: string;

  @Input() queryGroupKey: string;
  @Input() queryItemKey: string;

  @Input() set groups(value: SelectorGroup[]) {
    this._groups = value;
    if (this.groups && this.groups.length > 0 && this.groups[0].items && this.groups[0].items.length > 0) {

      let selectedGroup = this.groups[0];
      let selectedValue = this.groups[0].items[0];

      if (this.queryGroupKey && this.queryGroupValue) {
        const foundGroup = this.groups.find(g => g.value === this.queryGroupValue);
        if (foundGroup) {
          selectedGroup = foundGroup;
        }
        this.queryGroupValue = '';
      }

      if (this.queryItemKey && this.queryItemValue) {
        const foundValue = selectedGroup.items.find(i => i.value === this.queryItemValue);
        if (foundValue) {
          selectedValue = foundValue;
        }
        this.queryItemValue = '';
      }

      this.emitSelection(selectedGroup.value, selectedValue);
    }
  }

  @Output() selectionChanged: EventEmitter<{ group: string, item: SelectorItem }> = new EventEmitter();

  public selectedId: any;
  public selectedValue: SelectorItem;

  constructor(
    private router: Router,
    private activatedRoute: ActivatedRoute,
  ) { }

  ngOnInit(): void {
    this.queryGroupValue = this.activatedRoute.snapshot.queryParamMap.get(this.queryGroupKey);
    this.queryItemValue = this.activatedRoute.snapshot.queryParamMap.get(this.queryItemKey);
  }

  get groups(): SelectorGroup[] {
    return this._groups;
  }

  private emitSelection(group: string, item: SelectorItem) {
    this.selectedValue = item;
    this.selectedId = item.id;
    this.selectionChanged.emit({ group, item });

    // set group query
    const params: { key: string, value: string }[] = [];
    if (group && this.queryGroupKey) {
      params.push({ key: this.queryGroupKey, value: group });
    }

    // set item query
    if (item && this.queryItemKey) {
      params.push({ key: this.queryItemKey, value: item.value });
    }

    this.changeQueryParams(params);
  }

  public optionSelected(group: string, item: SelectorItem): void {
    this.emitSelection(group, item);
  }

  private changeQueryParams(p: { key: string, value: string }[]) {
    if (p) {
      const params: { [key: string]: string } = {};
      p.forEach(param => {
        params[param.key] = param.value;
      });

      this.router.navigate(['.'], {
        relativeTo: this.activatedRoute,
        queryParams: params,
        queryParamsHandling: 'merge',
      });
    }
  }

}
