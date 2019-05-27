import { BaseFactory } from './base-factory';
import {
  BorderConfig,
  CellAlignConfig,
  CellPaddingType,
  TableData,
  TableDesignType,
  TableRowHoverDesign,
  TableRowItem,
  TableRowMarkDesign,
} from '../model/tabledata';
import { DisplayedProduct } from '../../../../../models/product';
import { ElementRef } from '@angular/core';
import { PROVIDERS } from '../../../../../constants/providers';

export class ProductsListFactory {

  public static generateTableConfig(
    products: DisplayedProduct[],
    categoryRef: ElementRef,
    provider: string,
  ): TableData {

    const tableItems: TableRowItem[] = [];
    products.forEach((ds, index) => {
      const item = {
        config: {
          category: BaseFactory.generateStandardColumnBody(ds.category),
          type: BaseFactory.generateStandardColumnBody(ds.type),
          cpu: BaseFactory.generateStandardColumnBody(ds.cpu, '0', ' vCPUs', null, BaseFactory.generateNumberPipeConfig()),
          memory: BaseFactory.generateStandardColumnBody(ds.mem, '0', ' GB', null, BaseFactory.generateFixedNumberPipeConfig()),
          network: BaseFactory.generateStandardColumnBody(ds.ntwPerf),
          onDemand: BaseFactory.generateStandardColumnBody(
            ds.regularPrice,
            '',
            '$',
            '',
            BaseFactory.generateFixedNumberPipeConfig('5'),
          ),
          spotPrice: BaseFactory.generateStandardColumnBody(
            ds.spotPrice,
            '',
            '$',
            '',
            BaseFactory.generateFixedNumberPipeConfig('5'),
          ),
        },
        index: index,
      };

      if (provider !== PROVIDERS.amazon.provider && provider !== PROVIDERS.google.provider) {
        delete item.config.spotPrice;
      }

      tableItems.push(item);
    });

    const headers = {
      category: BaseFactory.generateTemplateColumnHeader(
        'category',
        categoryRef,
        false,
        BorderConfig.None,
        CellAlignConfig.Center,
        '80px',
        CellPaddingType.LeftNull,
      ),
      type: BaseFactory.generateStandardColumnHeader('machine type', '', BorderConfig.None, CellPaddingType.Left),
      cpu: BaseFactory.generateStandardColumnHeader('CPUs'),
      memory: BaseFactory.generateStandardColumnHeader('memory'),
      network: BaseFactory.generateStandardColumnHeader('network'),
      onDemand: BaseFactory.generateStandardColumnHeader('on-demand-price (Linux)', '250px'),
      spotPrice: BaseFactory.generateStandardColumnHeader('avg. spot price', '250px'),
    };

    if (provider !== PROVIDERS.amazon.provider && provider !== PROVIDERS.google.provider) {
      delete headers.spotPrice;
    }

    return {
      headers: headers,
      items: tableItems,
      designType: {
        type: TableDesignType.CardWithTemplateFirst,
        rowDesign: {
          hover: TableRowHoverDesign.White,
          mark: TableRowMarkDesign.None,
        },
      },
    };

  }

}
