import { Pipe, PipeTransform } from '@angular/core';
import { ProductCategory } from '../model/ProductCategory';

// Usage in template: <img src="assets/images/logos/{{ data.cloud | providerLogo }}">
@Pipe({
  name: 'categoryIcon',
})
export class CategoryIconPipe implements PipeTransform {

  transform(category: string): string {
    let categoryIcon = '';
    switch (category) {
      case ProductCategory.COMPUTE:
        categoryIcon = 'compute';
        break;
      case ProductCategory.GPU:
        categoryIcon = 'gpu';
        break;
      case ProductCategory.MEMORY:
        categoryIcon = 'memory';
        break;
      case ProductCategory.STORAGE:
        categoryIcon = 'storage';
        break;
      default:
        categoryIcon = 'general';
        break;
    }

    return `assets/images/ic_instance_category_${categoryIcon}.svg`;
  }
}
