import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'toFixedNumber'
})
export class ToFixedNumberPipe implements PipeTransform {

  transform(value: string, args?: any): any {

    if (args) {
      return Number(value).toFixed(args);
    }
    return Number(value).toFixed(2);

  }

}
