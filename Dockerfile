FROM php:8.3-apache

ENV PORT=10000 \
    APACHE_DOCUMENT_ROOT=/var/www/html/public

RUN apt-get update \
    && apt-get install -y --no-install-recommends libzip-dev libonig-dev unzip \
    && docker-php-ext-install mbstring zip \
    && docker-php-ext-enable opcache \
    && a2enmod rewrite headers expires \
    && sed -ri 's!/var/www/html!/var/www/html/public!g' /etc/apache2/sites-available/*.conf \
    && sed -ri 's!<Directory /var/www/>!<Directory /var/www/html/public/>!g' /etc/apache2/apache2.conf \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /var/www/html
COPY . /var/www/html

RUN mkdir -p storage/cache storage/logs storage/demo-documents \
    && chown -R www-data:www-data storage \
    && find storage -type d -exec chmod 775 {} \;

RUN printf '%s\n' \
    'opcache.enable=1' \
    'opcache.validate_timestamps=0' \
    'opcache.memory_consumption=128' \
    'upload_max_filesize=10M' \
    'post_max_size=12M' \
    'max_execution_time=120' \
    'memory_limit=256M' \
    > /usr/local/etc/php/conf.d/livira.ini

RUN sed -ri 's/^Listen 80$/Listen 10000/' /etc/apache2/ports.conf \
    && sed -ri 's/<VirtualHost \*:80>/<VirtualHost *:10000>/' /etc/apache2/sites-available/000-default.conf

EXPOSE 10000
CMD ["apache2-foreground"]
