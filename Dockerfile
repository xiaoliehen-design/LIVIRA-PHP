FROM php:8.3-apache

ENV PORT=10000 \
    APACHE_DOCUMENT_ROOT=/var/www/html/public

RUN apt-get update \
    && apt-get install -y --no-install-recommends libzip-dev libonig-dev unzip \
    && docker-php-ext-install mbstring zip \
    && docker-php-ext-enable opcache \
    && a2enmod rewrite headers expires \
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

# Render routes every request through Apache. This virtual host explicitly
# enables front-controller routing so /login, /inventory, /healthz, etc.
# are handled by public/index.php instead of returning Apache 404 responses.
RUN printf '%s\n' \
    '<VirtualHost *:10000>' \
    '    ServerName localhost' \
    '    DocumentRoot /var/www/html/public' \
    '    DirectoryIndex index.php' \
    '' \
    '    <Directory /var/www/html/public>' \
    '        Options FollowSymLinks' \
    '        AllowOverride All' \
    '        Require all granted' \
    '        FallbackResource /index.php' \
    '    </Directory>' \
    '' \
    '    ErrorLog ${APACHE_LOG_DIR}/error.log' \
    '    CustomLog ${APACHE_LOG_DIR}/access.log combined' \
    '</VirtualHost>' \
    > /etc/apache2/sites-available/000-default.conf \
    && printf '%s\n' 'ServerName localhost' > /etc/apache2/conf-available/livira-servername.conf \
    && a2enconf livira-servername \
    && sed -ri 's/^Listen 80$/Listen 10000/' /etc/apache2/ports.conf

EXPOSE 10000
CMD ["apache2-foreground"]
