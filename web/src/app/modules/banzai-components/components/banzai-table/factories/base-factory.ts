import {
  BorderConfig,
  CellAlignConfig,
  CellPaddingType,
  CellPositionType,
  CellType,
  PipeConfig,
  PipeType,
  TableDesignConfig,
  TableDesignType,
  TableHeaderConfig,
  TableRowConfig,
  TableRowHoverDesign,
  TableRowMarkDesign,
  TooltipConfig,
} from '../model/tabledata';
import { ElementRef } from '@angular/core';

export class BaseFactory {

  // --------- Headers --------- //

  public static generateStatusBarColumnHeader(): TableHeaderConfig {
    return {
      display: {},
      type: CellType.StatusBar,
      border: BorderConfig.None,
      disableSorting: true,
    };
  }

  public static generateStatusColumnHeader(): TableHeaderConfig {
    return {
      display: { header: 'status' },
      type: CellType.Status,
      border: BorderConfig.None,
    };
  }

  public static generateLogoColumnHeader(template: ElementRef, customWidth?: string,
                                         header?: string, cellAlign?: CellAlignConfig, disableSorting: boolean = true): TableHeaderConfig {
    return {
      display: {
        header: header,
      },
      type: CellType.Logo,
      logoRef: template,
      border: BorderConfig.None,
      cellAlignConfig: cellAlign,
      disableSorting: disableSorting,
      customWidth: customWidth,
    };
  }

  public static generateTemplateColumnHeader(
    headerTitle: string,
    template: ElementRef,
    disableSorting: boolean = false,
    border: BorderConfig = BorderConfig.None,
    cellAlign: CellAlignConfig = CellAlignConfig.Center,
    customWidth: string = null,
    cellPaddingConfig: CellPaddingType = null,
    cellPositionType: CellPositionType = null,
    subHeader: string = null,
  ): TableHeaderConfig {
    return {
      display: { header: headerTitle, subHeader: subHeader },
      type: CellType.Template,
      border: border,
      cellAlignConfig: cellAlign,
      templateRef: template,
      disableSorting: disableSorting,
      customWidth: customWidth,
      cellPaddingConfig: cellPaddingConfig,
      cellPositionConfig: cellPositionType,
    };
  }

  public static generateStandardColumnHeader(headerTitle: string, customWith?: string, border: BorderConfig = BorderConfig.None,
                                             paddingConfig: CellPaddingType = null): TableHeaderConfig {
    return {
      display: { header: headerTitle },
      type: CellType.Standard,
      border: border,
      cellPaddingConfig: paddingConfig,
      customWidth: customWith,
    };
  }

  public static generateStandardColumnHeaderDisabledSorting(headerTitle: string, subHeader: string = '',
                                                            borderConfig: BorderConfig = BorderConfig.None,
                                                            paddingConfig: CellPaddingType = null): TableHeaderConfig {
    return {
      display: { header: headerTitle, subHeader: subHeader },
      type: CellType.Standard,
      border: borderConfig,
      cellPaddingConfig: paddingConfig,
      disableSorting: true,
    };
  }

  public static generateMoreColumnHeader(template: ElementRef): TableHeaderConfig {
    return {
      display: { header: 'more' },
      customWidth: '56px',
      type: CellType.More,
      moreRef: template,
      border: BorderConfig.None,
      disableSorting: true,
    };
  }

  // --------- Bodies --------- //

  public static generateEmptyColumnBody(): TableRowConfig {
    return new TableRowConfig();
  }

  public static generateStandardColumnBody(
    value: any, defaultValue: any = '', postFix: any = null, label: any = null, displayPipeConfig: PipeConfig = null,
    tooltipConfig: TooltipConfig = null, enabledOverflow: boolean = false): TableRowConfig {
    return new TableRowConfig(
      {
        value: value,
        defaultValue: defaultValue,
        pipeConfig: displayPipeConfig,
        label: label,
        postFix: postFix,
        enabledOverflow: enabledOverflow,
      },
      tooltipConfig,
    );
  }

  public static generateStandardColumnBodyWithOverflow(value: any, tooltipConfig: TooltipConfig = null): TableRowConfig {
    return this.generateStandardColumnBody(value, null, null, null, null, tooltipConfig, true);
  }

  public static generateTooltipConfig(tooltip: string, pipeConfig: PipeConfig = null): TooltipConfig {
    return {
      tooltip: tooltip,
      pipeConfig: pipeConfig,
    };
  }

  // --------- Pipes --------- //

  public static generateTimeAgoPipeConfig(): PipeConfig {
    return {
      pipe: PipeType.TimeAgo,
    };
  }

  public static generateDatePipeConfig(format: string = 'MMM dd, y, H:mm'): PipeConfig {
    return {
      pipe: PipeType.Date,
      format: format,
    };
  }

  public static generateNumberPipeConfig(format: string = '1.0-2'): PipeConfig {
    return {
      pipe: PipeType.Number,
      format: format,
    };
  }

  public static generateFixedNumberPipeConfig(format: string = '2'): PipeConfig {
    return {
      pipe: PipeType.ToFixedNumber,
      format: format,
    };
  }

  public static generateTruncateAtMiddlePipeConfig(format: string = '20'): PipeConfig {
    return {
      pipe: PipeType.TruncateAtMiddle,
      format,
    };
  }

  // --------- Design --------- //

  public static generateDesignConfig(
    designType: TableDesignType = TableDesignType.Card,
    hoverDesign: TableRowHoverDesign = TableRowHoverDesign.None,
    markDesign: TableRowMarkDesign = TableRowMarkDesign.None,
  ): TableDesignConfig {
    return {
      type: designType,
      rowDesign: {
        hover: hoverDesign,
        mark: markDesign,
      },
    };
  }

}
