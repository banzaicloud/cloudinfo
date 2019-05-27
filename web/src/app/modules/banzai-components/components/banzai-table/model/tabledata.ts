import { ElementRef } from '@angular/core';

export interface TableData {
  headers: { [key: string]: TableHeaderConfig };
  items: TableRowItem[];
  detailsRef?: ElementRef;
  designType: TableDesignConfig;
  stopReRender?: boolean;
}

export interface TableHeaderConfig {
  display: TableHeaderDisplayConfig;
  type: CellType;
  logoRef?: ElementRef;
  moreRef?: ElementRef;
  border: BorderConfig;
  cellAlignConfig?: CellAlignConfig;
  cellPaddingConfig?: CellPaddingType;
  cellPositionConfig?: CellPositionType;
  templateRef?: ElementRef;
  disableSorting?: boolean;
  customWidth?: string;
}

export interface TableHeaderDisplayConfig {
  header?: string;
  subHeader?: string;
}

export interface TableRowItem {
  config: { [key: string]: TableRowConfig };
  selectedForMark?: boolean;
  index: number; // this is necessary because of table template changes
  groupName?: string; // if the items in groups, this is necessary for templates
}

export class TableRowConfig {
  constructor(public display?: {
                value: any,
                defaultValue?: any,
                label?: any, // this is for status values to display
                pipeConfig?: PipeConfig,
                postFix?: any,
                enabledOverflow?: boolean,
              },
              public tooltipConfig?: TooltipConfig) { }

  public getValue(): string {
    const display = this.display;
    return display ? (display.value ? display.value : (display.defaultValue ? display.defaultValue : '')) : '';
  }

  public getValueForStatus() {
    const display = this.display;
    return display ? (display.label ? display.label : this.getValue()) : '';
  }

}

export interface TooltipConfig {
  tooltip: string;
  pipeConfig: PipeConfig;
}

export interface PipeConfig {
  pipe: PipeType;
  format?: string;
}

export interface TableDesignConfig {
  type: TableDesignType;
  rowDesign: TableRowDesignConfig;
}

export interface TableRowDesignConfig {
  hover: TableRowHoverDesign;
  mark: TableRowMarkDesign;
}

export enum CellType {
  StatusBar = 'statusBar',
  Status = 'status',
  Standard = 'standard',
  Logo = 'logo',
  Template = 'template',
  More = 'more',
}

export enum BorderConfig {
  Left = 'left',
  Right = 'right',
  Both = 'edge',
  None = 'none',
}

export enum CellAlignConfig {
  Center = 'center',
}

export enum PipeType {
  TimeAgo = 'timeAgo',
  Date = 'date',
  Number = 'number',
  TruncateAtMiddle = 'truncateAtMiddle',
  ToFixedNumber = 'toFixedNumber',
}

export enum CellPositionType {
  Relative = 'relative'
}

export enum CellPaddingType {
  Left = 'left',
  LeftNull = 'left-null',
  Right = 'right',
  Edges = 'edges',
}

export enum TableDesignType {
  Card = 'card',
  CardWithTemplateFirst = 'cardFirstTemplate',
  Border = 'border',
  Base = 'base',
  BaseWithCheckbox = 'baseWithCheckbox',
}

export enum TableRowHoverDesign {
  White = 'white',
  None = 'none',
}

export enum TableRowMarkDesign {
  LightRed = 'lightRed',
  Smoke = 'smoke',
  None = 'none',
}
