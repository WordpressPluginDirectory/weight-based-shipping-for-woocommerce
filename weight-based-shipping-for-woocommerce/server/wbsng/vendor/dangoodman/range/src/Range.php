<?php
namespace GzpWbsNgVendors\Dgm\Range;

use GzpWbsNgVendors\Dgm\SimpleProperties\SimpleProperties;
use GzpWbsNgVendors\Dgm\Comparator\IComparator;


/**
 * @property-read mixed $min
 * @property-read mixed $max
 * @property-read bool $minInclusive
 * @property-read bool $maxInclusive
 */
class Range extends SimpleProperties
{
    public function __construct($min, $max, $minInclusive = true, $maxInclusive = true)
    {
        $this->min = $min;
        $this->max = $max;
        $this->minInclusive = (bool)$minInclusive;
        $this->maxInclusive = (bool)$maxInclusive;
    }

    public function clamp($value)
    {
        if (isset($this->min)) {
            $value = max($this->min, $value);
        }

        if (isset($this->max)) {
            $value = min($this->max, $value);
        }

        return $value;
    }

    public function includes($value, IComparator $comparator)
    {
        return
            (!isset($this->min) || $comparator->greater($value, $this->min, $this->minInclusive)) &&
            (!isset($this->max) || $comparator->less($value, $this->max, $this->maxInclusive));
    }

    protected $min;
    protected $max;
    protected $minInclusive;
    protected $maxInclusive;
}