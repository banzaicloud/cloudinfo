import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'truncateAtMiddle'
})
export class TruncateAtMiddlePipe implements PipeTransform {

  transform(value: any, maxLengthForTruncate: string | number): any {
    if (value.length > maxLengthForTruncate) {
      return value.slice(0, Number(maxLengthForTruncate) / 2)
        .concat('...')
        .concat(value.slice(value.length - Number(maxLengthForTruncate) / 2));
    } else {
      return value;
    }
  }

}
